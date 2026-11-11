(function () {
    'use strict';

    // ---------- Shared helpers ----------
    function mark(input, ok) {
        input.classList.toggle('is-valid', ok);
        input.classList.toggle('is-invalid', !ok);
        return ok;
    }

    function clearMark(input) {
        input.classList.remove('is-valid', 'is-invalid');
    }

    // ============================================================
    // LOGIN PAGE
    // ============================================================
    (function initLogin() {
        const form = document.getElementById('loginForm');
        if (!form) return;

        const email         = document.getElementById('loginEmail');
        const password      = document.getElementById('loginPassword');
        const pwToggle      = document.getElementById('loginPwToggle');
        const emailFeedback = email.parentElement.querySelector('.invalid-feedback');
        const problemLink    = document.getElementById('passwordProblemLink');
        const loginAlert = document.getElementById('loginAlert');

        function showLoginAlert(message, type) {
            loginAlert.textContent = message;
            loginAlert.className = `alert alert-${type} mb-3`;
            loginAlert.classList.remove('d-none');
            loginAlert.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }

        function hideLoginAlert() {
            loginAlert.classList.add('d-none');
            loginAlert.textContent = '';
        }

        function validateEmail() {
            const v = email.value.trim();
            const ok = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(v) && v.length <= 254;
            emailFeedback.textContent = ok
                ? ''
                : (v.length === 0
                    ? 'Please enter your email address.'
                    : 'Please enter a valid email address.');
            return mark(email, ok);
        }

        function validatePassword() {
            return mark(password, password.value.length > 0);
        }

        const touched = new WeakSet();
        function bindLive(input, validator) {
            input.addEventListener('blur', () => {
                touched.add(input);
                validator();
            });
            input.addEventListener('input', () => {
                if (touched.has(input)) validator();
            });
        }

        bindLive(email, validateEmail);
        bindLive(password, validatePassword);

        [email, password].forEach(input => {
            input.addEventListener('input', () => {
                if (input.value.length === 0 && !touched.has(input)) clearMark(input);
            });
        });

        pwToggle.addEventListener('click', () => {
            const show = password.type === 'password';
            password.type = show ? 'text' : 'password';
            pwToggle.setAttribute('aria-pressed', String(show));
            pwToggle.setAttribute('aria-label', show ? 'Hide password' : 'Show password');
            pwToggle.querySelector('i').className = show ? 'bi bi-eye-slash' : 'bi bi-eye';
        });

        problemLink.addEventListener('click', (e) => {
            e.preventDefault();
            const dlg = document.getElementById('passwordHelpDialog');
            if (!dlg) return;

            const dialogEmail = document.getElementById('passwordHelpEmail');
            const feedback    = document.getElementById('passwordHelpFeedback');
            const alertEl     = document.getElementById('passwordHelpAlert');

            dialogEmail.value = email.value.trim();
            dialogEmail.classList.remove('is-valid', 'is-invalid');
            feedback.textContent = '';
            feedback.style.display = 'none';
            alertEl.classList.add('d-none');
            alertEl.textContent = '';

            dlg.showModal();
            setTimeout(() => dialogEmail.focus(), 100);
        });

        // ============================================================
        // PASSWORD HELP DIALOG (login page only)
        // ============================================================
        (function initPasswordHelpDialog() {
            const dialog = document.getElementById('passwordHelpDialog');
            if (!dialog) return;

            const dialogEmail      = document.getElementById('passwordHelpEmail');
            const feedback         = document.getElementById('passwordHelpFeedback');
            const alertEl          = document.getElementById('passwordHelpAlert');
            const forgottenBtn     = document.getElementById('forgottenPasswordBtn');
            const expiredBtn       = document.getElementById('expiredPasswordBtn');

            function showAlert(message, type) {
                alertEl.textContent = message;
                alertEl.className = `alert alert-${type} mb-0`;
                alertEl.classList.remove('d-none');
            }
            function hideAlert() {
                alertEl.classList.add('d-none');
                alertEl.textContent = '';
            }
            function validateEmail() {
                const v = dialogEmail.value.trim();
                const ok = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(v) && v.length <= 254;
                dialogEmail.classList.toggle('is-valid', ok);
                dialogEmail.classList.toggle('is-invalid', !ok);
                feedback.textContent = ok ? '' :
                    (v.length === 0 ? 'Please enter your email address.' : 'Please enter a valid email address.');
                feedback.style.display = ok ? 'none' : 'block';
                return ok;
            }

            // Close buttons (X and Cancel)
            dialog.querySelectorAll('[data-dialog-close]').forEach(btn => {
                btn.addEventListener('click', () => dialog.close('cancel'));
            });

            // Backdrop click
            dialog.addEventListener('click', (e) => {
                const r = dialog.getBoundingClientRect();
                const outside =
                    e.clientX < r.left || e.clientX > r.right ||
                    e.clientY < r.top  || e.clientY > r.bottom;
                if (outside) dialog.close('cancel');
            });

            // Live validation on blur and re-validation on input if already invalid
            dialogEmail.addEventListener('blur', validateEmail);
            dialogEmail.addEventListener('input', () => {
                if (dialogEmail.classList.contains('is-invalid')) validateEmail();
            });

            async function sendMagicLink(btn, reasonLabel) {
                hideAlert();
                if (!validateEmail()) {
                    dialogEmail.focus();
                    return;
                }

                const originalText = btn.innerHTML;
                const otherBtn     = btn === forgottenBtn ? expiredBtn : forgottenBtn;
                btn.disabled = true;
                otherBtn.disabled = true;
                btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Sending…';

                try {
                    const res = await fetch('/api/auth/magic-link/request', {
                        method:  'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body:    JSON.stringify({ email: dialogEmail.value.trim() })
                    });
                    const data = await res.json().catch(() => ({}));

                    if (!res.ok) {
                        showAlert(data.message || `Request failed (HTTP ${res.status}).`, 'danger');
                        return;
                    }

                    showAlert(
                        data.message ||
                        `If an account exists for that email, we've sent you a sign-in link. Check your inbox.`,
                        'success'
                    );
                } catch {
                    showAlert('Network error. Please check your connection and try again.', 'danger');
                } finally {
                    btn.disabled = false;
                    otherBtn.disabled = false;
                    btn.innerHTML = originalText;
                }
            }

            forgottenBtn.addEventListener('click', () => sendMagicLink(forgottenBtn, 'forgotten'));
            expiredBtn.addEventListener('click',   () => sendMagicLink(expiredBtn,   'expired'));
        })();

        form.addEventListener('submit', async function (e) {
            e.preventDefault();
            hideLoginAlert();

            const okEmail = validateEmail();
            const okPw    = validatePassword();
            if (!okEmail || !okPw) {
                const firstInvalid = form.querySelector('.is-invalid');
                if (firstInvalid) {
                    firstInvalid.focus();
                    firstInvalid.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
                return;
            }

            const submitBtn = form.querySelector('button[type="submit"]');
            const originalText = submitBtn.textContent;
            submitBtn.disabled = true;
            submitBtn.textContent = 'Signing in…';

            try {
                const res = await fetch('/api/login', {
                    method:  'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body:    JSON.stringify({
                        email:    email.value.trim(),
                        password: password.value
                    })
                });

                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    if (res.status === 401 || data.code === 16) {
                        showLoginAlert('Invalid email or password.', 'danger');
                    } else if (data.code === 9) {
                        showLoginAlert('Your password expired or account is not verified after registration.', 'warning');
                    } else if (res.status === 400 || data.code === 3) {
                        showLoginAlert(data.message || 'Please enter your email and password.', 'danger');
                    } else {
                        showLoginAlert(data.message || `Sign-in failed (HTTP ${res.status}).`, 'danger');
                    }
                    return;
                }

                localStorage.setItem('token', data.token);
                if (data.user) {
                    localStorage.setItem('user', JSON.stringify(data.user));
                }

                showLoginAlert('Signed in. Redirecting…', 'success');
                setTimeout(() => {
                    window.location.href = '/HTML/dashboard.html';
                }, 800);

            } catch (err) {
                showLoginAlert('Network error. Please check your connection and try again.', 'danger');
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = originalText;
            }
        });
    })();

    // ============================================================
    // REGISTER PAGE
    // ============================================================
    (function initRegister() {
        const form = document.getElementById('registerForm');
        if (!form) return;

        const firstName = document.getElementById('firstName');
        const lastName  = document.getElementById('lastName');
        const email     = document.getElementById('email');
        const password  = document.getElementById('password');
        const confirm   = document.getElementById('confirmPassword');
        const terms     = document.getElementById('terms');

        const ruleList      = document.getElementById('ruleList');
        const strengthFill  = document.getElementById('strengthFill');
        const strengthLabel = document.getElementById('strengthLabel');
        const emailFeedback = document.getElementById('emailFeedback');

        const registerAlert = document.getElementById('registerAlert');

        function showAlert(message, type) {
            registerAlert.textContent = message;
            registerAlert.className = `alert alert-${type} mb-3`;
            registerAlert.classList.remove('d-none');
        }

        function hideAlert() {
            registerAlert.classList.add('d-none');
            registerAlert.textContent = '';
        }

        function validateName(input) {
            const v = input.value.trim();
            const ok = v.length >= 2 && v.length <= 50 && /^[\p{L}][\p{L}\s'’.-]*$/u.test(v);
            return mark(input, ok);
        }

        function validateEmail() {
            const v = email.value.trim();
            const ok = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(v) && v.length <= 254;
            emailFeedback.textContent = ok
                ? ''
                : (v.length === 0
                    ? 'Please enter your email address.'
                    : 'Please enter a valid email address.');
            return mark(email, ok);
        }

        const RULES = {
            length : v => v.length >= 12,
            upper  : v => /[A-Z]/.test(v),
            lower  : v => /[a-z]/.test(v),
            digit  : v => /[0-9]/.test(v),
            special: v => /[^A-Za-z0-9]/.test(v)
        };

        function passwordMeetsAll(v) {
            return Object.keys(RULES).every(k => RULES[k](v));
        }

        function validatePassword() {
            return mark(password, passwordMeetsAll(password.value));
        }

        function validateConfirm() {
            return mark(confirm, confirm.value.length > 0 && confirm.value === password.value);
        }

        function validateTerms() {
            const ok = terms.checked;
            terms.classList.toggle('is-invalid', !ok);
            return ok;
        }

        function updatePasswordUI() {
            const v = password.value;
            let met = 0;
            ruleList.querySelectorAll('li').forEach(li => {
                const rule = li.getAttribute('data-rule');
                const ok = RULES[rule](v);
                if (ok) met++;
                li.classList.toggle('ok', ok);
                li.querySelector('i').className = ok ? 'bi bi-check-circle-fill' : 'bi bi-circle';
            });
            strengthFill.style.width = ((met / 5) * 100) + '%';
            let label = 'Password strength';
            let color = '#e2e8f0';
            if (v.length > 0) {
                if (met <= 2)      { label = 'Weak';   color = '#dc3545'; }
                else if (met <= 3) { label = 'Fair';   color = '#f59e0b'; }
                else if (met === 4){ label = 'Good';   color = '#0ea5e9'; }
                else               { label = 'Strong'; color = '#16a34a'; }
            }
            strengthFill.style.background = color;
            strengthLabel.textContent = label;
            if (confirm.value.length > 0) validateConfirm();
        }

        const touched = new WeakSet();
        function bindLive(input, validator) {
            input.addEventListener('blur', () => {
                touched.add(input);
                validator();
            });
            input.addEventListener('input', () => {
                if (touched.has(input)) validator();
            });
        }

        bindLive(firstName, () => validateName(firstName));
        bindLive(lastName,  () => validateName(lastName));
        bindLive(email,     validateEmail);
        bindLive(password,  validatePassword);
        bindLive(confirm,   validateConfirm);

        password.addEventListener('input', updatePasswordUI);
        password.addEventListener('blur', updatePasswordUI);

        terms.addEventListener('change', () => { touched.add(terms); validateTerms(); });

        function bindToggle(buttonId, input) {
            const btn = document.getElementById(buttonId);
            btn.addEventListener('click', () => {
                const show = input.type === 'password';
                input.type = show ? 'text' : 'password';
                btn.setAttribute('aria-pressed', String(show));
                btn.setAttribute('aria-label', show ? 'Hide password' : 'Show password');
                btn.querySelector('i').className = show ? 'bi bi-eye-slash' : 'bi bi-eye';
            });
        }
        bindToggle('pwToggle', password);
        bindToggle('pwToggle2', confirm);

        form.addEventListener('submit', async function (e) {
            e.preventDefault();
            hideAlert();

            const results = [
                validateName(firstName),
                validateName(lastName),
                validateEmail(),
                validatePassword(),
                validateConfirm(),
                validateTerms()
            ];
            updatePasswordUI();
            if (results.includes(false)) {
                const firstInvalid = form.querySelector('.is-invalid');
                if (firstInvalid) {
                    firstInvalid.focus();
                    firstInvalid.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
                return;
            }

            const submitBtn = form.querySelector('button[type="submit"]');
            const originalText = submitBtn.textContent;
            submitBtn.disabled = true;
            submitBtn.textContent = 'Creating account…';

            try {
                const payload = {
                    email:     email.value.trim(),
                    password:  password.value,
                    firstName: firstName.value.trim(),
                    lastName:  lastName.value.trim()
                };

                const res = await fetch('/api/register', {
                    method:  'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body:    JSON.stringify(payload)
                });

                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    if (res.status === 409 || data.code === 6) {
                        email.classList.add('is-invalid');
                        email.classList.remove('is-valid');
                        emailFeedback.textContent = 'An account with this email already exists.';
                        email.focus();
                        email.scrollIntoView({ behavior: 'smooth', block: 'center' });
                    } else if (res.status === 400 || data.code === 3) {
                        showAlert(data.message || 'Please check your input and try again.', 'danger');
                    } else {
                        showAlert(data.message || `Registration failed (HTTP ${res.status}).`, 'danger');
                    }
                    return;
                }

                showAlert(data.message || 'User registration successful. Redirecting…', 'success');
                setTimeout(() => { window.location.href = 'login.html'; }, 1200);

            } catch (err) {
                showAlert('Network error. Please check your connection and try again.', 'danger');
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = originalText;
            }
        });

        [firstName, lastName, email, password, confirm].forEach(input => {
            input.addEventListener('input', () => {
                if (input.value.length === 0 && !touched.has(input)) clearMark(input);
            });
        });

        updatePasswordUI();
    })();

    // ============================================================
    // EMAIL VERIFICATION PAGE
    // ============================================================
    (function initVerifyEmail() {
        const elLoading = document.getElementById('state-loading');
        if (!elLoading) return;

        const elSuccess = document.getElementById('state-success');
        const elError   = document.getElementById('state-error');
        const elNoToken = document.getElementById('state-no-token');

        function show(el) {
            [elLoading, elSuccess, elError, elNoToken].forEach(x => x.classList.add('d-none'));
            el.classList.remove('d-none');
        }

        const params = new URLSearchParams(window.location.search);
        const token  = params.get('token');

        if (!token) {
            show(elNoToken);
            return;
        }

        fetch('/api/verify/' + encodeURIComponent(token))
            .then(async (res) => {
                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    document.getElementById('error-message').textContent =
                        data.message || `Verification failed (HTTP ${res.status}).`;
                    show(elError);
                    return;
                }

                if (data.message) {
                    document.getElementById('success-message').textContent = data.message;
                }
                show(elSuccess);
            })
            .catch(() => {
                document.getElementById('error-message').textContent =
                    'Network error. Please check your connection and try again.';
                show(elError);
            });
    })();

    // ============================================================
// LOGOUT CONFIRMATION DIALOG (dashboard only)
// ============================================================
    (function initLogoutDialog() {
        const trigger = document.getElementById('logoutTrigger');
        const dialog  = document.getElementById('logoutDialog');
        if (!trigger || !dialog) return;

        const confirmBtn = document.getElementById('logoutConfirm');

        // Open
        trigger.addEventListener('click', (e) => {
            e.preventDefault();
            dialog.showModal();
        });

        // Cancel via the X button and the Cancel button
        dialog.querySelectorAll('[data-dialog-close]').forEach((btn) => {
            btn.addEventListener('click', () => dialog.close('cancel'));
        });

        // Backdrop click
        dialog.addEventListener('click', (e) => {
            const r = dialog.getBoundingClientRect();
            const outside =
                e.clientX < r.left || e.clientX > r.right ||
                e.clientY < r.top  || e.clientY > r.bottom;
            if (outside) dialog.close('cancel');
        });

        // Confirm
        confirmBtn.addEventListener('click', () => dialog.close('confirm'));

        // Fire the actual redirect only on confirm
        dialog.addEventListener('close', () => {
            if (dialog.returnValue === 'confirm') {
                localStorage.removeItem('token');
                window.location.href = 'login.html';
            }
        });
    })();

    // ============================================================
    // MAGIC LINK LOGIN PAGE
    // ============================================================
    (function initMagicLogin() {
        const elLoading = document.getElementById('magic-loading');
        if (!elLoading) return;

        const elSuccess = document.getElementById('magic-success');
        const elError   = document.getElementById('magic-error');
        const elNoToken = document.getElementById('magic-no-token');

        function show(el) {
            [elLoading, elSuccess, elError, elNoToken].forEach(x => x && x.classList.add('d-none'));
            el.classList.remove('d-none');
        }

        const token = new URLSearchParams(window.location.search).get('token');

        if (!token) {
            show(elNoToken);
            return;
        }

        fetch('/api/auth/magic-link/verify', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ token })
        })
            .then(async (res) => {
                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    document.getElementById('magic-error-message').textContent =
                        data.message || `Sign-in failed (HTTP ${res.status}).`;
                    show(elError);
                    return;
                }

                localStorage.setItem('token', data.token);
                if (data.user) {
                    localStorage.setItem('user', JSON.stringify(data.user));
                }

                show(elSuccess);
                setTimeout(() => { window.location.replace('dashboard.html'); }, 300);
            })
            .catch(() => {
                document.getElementById('magic-error-message').textContent =
                    'Network error. Please check your connection and try again.';
                show(elError);
            });
    })();
})();