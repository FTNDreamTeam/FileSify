(function () {
    'use strict';
    // Current folder path
    let currentFolderPath = '';

    // ---------- Shared helpers ----------
    function mark(input, ok) {
        input.classList.toggle('is-valid', ok);
        input.classList.toggle('is-invalid', !ok);
        return ok;
    }

    function clearMark(input) {
        input.classList.remove('is-valid', 'is-invalid');
    }

    // ---------- Email validation ----------
    const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;
    function emailError(value) {
        const v = (value || '').trim();
        if (v.length === 0) return 'Please enter your email address.';
        if (v.length > 254 || !EMAIL_RE.test(v)) return 'Please enter a valid email address.';
        return '';
    }

    // ---------- Password rules ----------
    const PASSWORD_RULES = {
        length: v => v.length >= 12,
        upper: v => /[A-Z]/.test(v),
        lower: v => /[a-z]/.test(v),
        digit: v => /[0-9]/.test(v),
        special: v => /[^A-Za-z0-9]/.test(v)
    };
    function passwordMeetsAll(v) {
        return Object.keys(PASSWORD_RULES).every(k => PASSWORD_RULES[k](v));
    }
    function computeStrength(value) {
        let met = 0;
        for (const k of Object.keys(PASSWORD_RULES)) if (PASSWORD_RULES[k](value)) met++;
        let label = 'Password strength';
        let color = '#e2e8f0';
        if (value.length > 0) {
            if (met <= 2) { label = 'Weak'; color = '#dc3545'; }
            else if (met <= 3) { label = 'Fair'; color = '#f59e0b'; }
            else if (met === 4) { label = 'Good'; color = '#0ea5e9'; }
            else { label = 'Strong'; color = '#16a34a'; }
        }
        return { met, label, color };
    }
    function renderStrength(ruleListEl, fillEl, labelEl, value) {
        const { met, label, color } = computeStrength(value);
        ruleListEl.querySelectorAll('li').forEach(li => {
            const rule = li.getAttribute('data-rule');
            const ok = PASSWORD_RULES[rule](value);
            li.classList.toggle('ok', ok);
            li.querySelector('i').className = ok ? 'bi bi-check-circle-fill' : 'bi bi-circle';
        });
        fillEl.style.width = ((met / 5) * 100) + '%';
        fillEl.style.background = color;
        labelEl.textContent = label;
    }

    // ---------- Alert helpers ----------
    function setAlert(el, message, type, extraClass) {
        el.textContent = message;
        el.className = `alert alert-${type} ${extraClass || 'mb-3'}`;
        el.classList.remove('d-none');
    }
    function clearAlert(el) {
        el.classList.add('d-none');
        el.textContent = '';
    }

    // ---------- JWT decoder ----------
    function decodeJwtPayload(jwt) {
        try {
            const parts = jwt.split('.');
            if (parts.length !== 3) return {};
            let b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
            while (b64.length % 4) b64 += '=';
            return JSON.parse(atob(b64));
        } catch { return {}; }
    }

    // ---------- Password toggle ----------
    function bindPasswordToggle(buttonId, input) {
        const btn = document.getElementById(buttonId);
        if (!btn) return;
        btn.addEventListener('click', () => {
            const show = input.type === 'password';
            input.type = show ? 'text' : 'password';
            btn.setAttribute('aria-pressed', String(show));
            btn.setAttribute('aria-label', show ? 'Hide password' : 'Show password');
            btn.querySelector('i').className = show ? 'bi bi-eye-slash' : 'bi bi-eye';
        });
    }

    // ---------- Live validation binder ----------
    function createLiveBinder() {
        const touched = new WeakSet();
        return function bindLive(input, validator) {
            input.addEventListener('blur', () => { touched.add(input); validator(); });
            input.addEventListener('input', () => { if (touched.has(input)) validator(); });
        };
    }

    // ---------- Dialog wiring ----------
    function wireDialogClose(dialog) {
        dialog.querySelectorAll('[data-dialog-close]').forEach(btn => {
            btn.addEventListener('click', () => dialog.close('cancel'));
        });
        dialog.addEventListener('click', (e) => {
            const r = dialog.getBoundingClientRect();
            const outside = e.clientX < r.left || e.clientX > r.right ||
                e.clientY < r.top || e.clientY > r.bottom;
            if (outside) dialog.close('cancel');
        });
    }

    // ---------- Login ----------
    (function initLogin() {
        // Login form definition
        const form = document.getElementById('loginForm');
        if (!form) return;

        const email = document.getElementById('loginEmail');
        const password = document.getElementById('loginPassword');
        const emailFeedback = email.parentElement.querySelector('.invalid-feedback');
        const problemLink = document.getElementById('passwordProblemLink');
        const loginAlert = document.getElementById('loginAlert');

        // Login alert
        function showLoginAlert(message, type) {
            setAlert(loginAlert, message, type);
            loginAlert.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }

        function hideLoginAlert() { clearAlert(loginAlert); }

        // Email validation
        function validateEmail() {
            const msg = emailError(email.value);
            emailFeedback.textContent = msg;
            return mark(email, msg === '');
        }

        // Password validation
        function validatePassword() {
            return mark(password, password.value.length > 0);
        }

        // Fill out watching
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

        // Validation on every change
        bindLive(email, validateEmail);
        bindLive(password, validatePassword);

        // Hide field errors on empty
        [email, password].forEach(input => {
            input.addEventListener('input', () => {
                if (input.value.length === 0 && !touched.has(input)) clearMark(input);
            });
        });

        // Password show and hide
        bindPasswordToggle('loginPwToggle', password);

        // Problem link click
        problemLink.addEventListener('click', (e) => {
            // Form submitting turn off
            e.preventDefault();

            // Problem form definition
            const dlg = document.getElementById('passwordHelpDialog');
            if (!dlg) return;

            const dialogEmail = document.getElementById('passwordHelpEmail');
            const feedback = document.getElementById('passwordHelpFeedback');
            const alertEl = document.getElementById('passwordHelpAlert');

            dialogEmail.value = email.value.trim();
            dialogEmail.classList.remove('is-valid', 'is-invalid');
            feedback.textContent = '';
            feedback.style.display = 'none';
            alertEl.classList.add('d-none');
            alertEl.textContent = '';

            dlg.showModal();
            setTimeout(() => dialogEmail.focus(), 100);
        });

        // Password help dialog
        (function initPasswordHelpDialog() {
            // Password help dialog element definition
            const dialog = document.getElementById('passwordHelpDialog');
            if (!dialog) return;

            const dialogEmail = document.getElementById('passwordHelpEmail');
            const feedback = document.getElementById('passwordHelpFeedback');
            const alertEl = document.getElementById('passwordHelpAlert');
            const forgottenBtn = document.getElementById('forgottenPasswordBtn');
            const expiredBtn = document.getElementById('expiredPasswordBtn');

            // Dialog alert
            function showAlert(message, type) { setAlert(alertEl, message, type, 'mb-0'); }
            function hideAlert() { clearAlert(alertEl); }

            // Email validation
            function validateEmail() {
                const msg = emailError(dialogEmail.value);
                feedback.textContent = msg;
                feedback.style.display = msg ? 'block' : 'none';
                return mark(dialogEmail, msg === '');
            }

            wireDialogClose(dialog);

            // Live validation on blur and re-validation on input if already invalid
            dialogEmail.addEventListener('blur', validateEmail);
            dialogEmail.addEventListener('input', () => {
                if (dialogEmail.classList.contains('is-invalid')) validateEmail();
            });

            // Magic link sending for forgotten pasword
            async function sendMagicLink(btn, reasonLabel) {
                hideAlert();
                if (!validateEmail()) {
                    dialogEmail.focus();
                    return;
                }

                const originalText = btn.innerHTML;
                const otherBtn = btn === forgottenBtn ? expiredBtn : forgottenBtn;
                btn.disabled = true;
                otherBtn.disabled = true;
                btn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Sending…';

                try {
                    const res = await fetch('/api/auth/magic-link/request', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ email: dialogEmail.value.trim() })
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

            // Forgotten password button
            forgottenBtn.addEventListener('click', () => sendMagicLink(forgottenBtn, 'forgotten'));

            // Expired password dialog open
            expiredBtn.addEventListener('click', () => {
                if (!validateEmail()) {
                    dialogEmail.focus();
                    return;
                }
                const emailValue = dialogEmail.value.trim();
                dialog.close('expired');

                const changeDlg = document.getElementById('changePasswordDialog');
                if (!changeDlg) return;

                // Pre-fill the email from dialog 1
                const cpEmail = document.getElementById('cpEmail');
                if (cpEmail) cpEmail.value = emailValue;

                changeDlg.showModal();
            });
        })();

        // Login form submit
        form.addEventListener('submit', async function (e) {
            // Login form element definition
            e.preventDefault();
            hideLoginAlert();

            // Validation
            const okEmail = validateEmail();
            const okPw = validatePassword();
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

            // Send login credentials
            try {
                const res = await fetch('/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        email: email.value.trim(),
                        password: password.value
                    })
                });

                // Response checking
                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    const raw = (data.message || '').toLowerCase();

                    if (res.status === 401 || data.code === 16) {
                        showLoginAlert('Invalid email or password.', 'danger');
                    } else if (data.code === 9) {
                        if (raw.includes('not verified')) {
                            showLoginAlert('Your email is not verified yet. Check your inbox for the verification link.', 'warning');
                        } else if (raw.includes('expired')) {
                            showLoginAlert('Your password has expired. Click "Problem with password?" below to reset it.', 'warning');
                        } else {
                            showLoginAlert('Sign-in blocked. Please check your account status.', 'warning');
                        }
                    } else if (raw.includes('60 seconds') || raw.includes('before requesting')) {
                        // Cooldown: OTP was just sent and not used
                        showLoginAlert(
                            'A verification code was just sent to your email. Because you did not use it, please wait 60 seconds and try signing in again.',
                            'warning'
                        );
                    } else if (res.status === 400 || data.code === 3) {
                        showLoginAlert(data.message || 'Please enter your email and password.', 'danger');
                    } else if (data.code === 13 || res.status >= 500) {
                        showLoginAlert('Server error. Please try again in a moment.', 'danger');
                    } else {
                        showLoginAlert('Sign-in failed. Please try again.', 'danger');
                    }
                    return;
                }

                // Checking for second verification need + redirect
                if (data.twoFactor === true) {
                    sessionStorage.setItem('pending_token', data.token);
                    sessionStorage.setItem('pending_email', email.value.trim());
                    window.location.href = '/HTML/two-factor.html';
                    return;
                }

                // Token storing
                localStorage.setItem('token', data.token);
                if (data.user) {
                    localStorage.setItem('user', JSON.stringify(data.user));
                }

                // Login succes message + redirect
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

    // ---------- Register ----------
    (function initRegister() {
        // Register form definition
        const form = document.getElementById('registerForm');
        if (!form) return;

        const firstName = document.getElementById('firstName');
        const lastName = document.getElementById('lastName');
        const email = document.getElementById('email');
        const password = document.getElementById('password');
        const confirm = document.getElementById('confirmPassword');
        const terms = document.getElementById('terms');

        const ruleList = document.getElementById('ruleList');
        const strengthFill = document.getElementById('strengthFill');
        const strengthLabel = document.getElementById('strengthLabel');
        const emailFeedback = document.getElementById('emailFeedback');

        const registerAlert = document.getElementById('registerAlert');

        // Register alert
        function showAlert(message, type) { setAlert(registerAlert, message, type); }
        function hideAlert() { clearAlert(registerAlert); }

        // Name validation
        function validateName(input) {
            const v = input.value.trim();
            const ok = v.length >= 2 && v.length <= 50 && /^[\p{L}][\p{L}\s'’.-]*$/u.test(v);
            return mark(input, ok);
        }

        // Email validation
        function validateEmail() {
            const msg = emailError(email.value);
            emailFeedback.textContent = msg;
            return mark(email, msg === '');
        }

        // Password validation
        function validatePassword() {
            return mark(password, passwordMeetsAll(password.value));
        }

        function validateConfirm() {
            return mark(confirm, confirm.value.length > 0 && confirm.value === password.value);
        }

        // Terms validation
        function validateTerms() {
            const ok = terms.checked;
            terms.classList.toggle('is-invalid', !ok);
            return ok;
        }

        // Password success bar
        function updatePasswordUI() {
            renderStrength(ruleList, strengthFill, strengthLabel, password.value);
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

        // Validation on every change
        bindLive(firstName, () => validateName(firstName));
        bindLive(lastName, () => validateName(lastName));
        bindLive(email, validateEmail);
        bindLive(password, validatePassword);
        bindLive(confirm, validateConfirm);

        // Password bar update on change
        password.addEventListener('input', updatePasswordUI);
        password.addEventListener('blur', updatePasswordUI);

        terms.addEventListener('change', () => { touched.add(terms); validateTerms(); });

        // Password show and hide
        bindPasswordToggle('pwToggle', password);
        bindPasswordToggle('pwToggle2', confirm);

        // Register form submit button
        form.addEventListener('submit', async function (e) {
            e.preventDefault();
            hideAlert();

            // Input error/validation check
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

            // Send register credentials
            try {
                const payload = {
                    email: email.value.trim(),
                    password: password.value,
                    firstName: firstName.value.trim(),
                    lastName: lastName.value.trim()
                };

                const res = await fetch('/api/register', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                // Response check
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

                // Succesfull message + redirect
                showAlert(data.message || 'User registration successful. Redirecting…', 'success');
                setTimeout(() => { window.location.href = 'login.html'; }, 1200);

            } catch (err) {
                showAlert('Network error. Please check your connection and try again.', 'danger');
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = originalText;
            }
        });

        // Hide field errors on empty
        [firstName, lastName, email, password, confirm].forEach(input => {
            input.addEventListener('input', () => {
                if (input.value.length === 0 && !touched.has(input)) clearMark(input);
            });
        });

        updatePasswordUI();
    })();

    // ---------- Email verification for registration ----------
    (function initVerifyEmail() {
        // Verification page element definition
        const elLoading = document.getElementById('state-loading');
        if (!elLoading) return;

        const elSuccess = document.getElementById('state-success');
        const elError = document.getElementById('state-error');
        const elNoToken = document.getElementById('state-no-token');

        function show(el) {
            [elLoading, elSuccess, elError, elNoToken].forEach(x => x.classList.add('d-none'));
            el.classList.remove('d-none');
        }

        const params = new URLSearchParams(window.location.search);
        const token = params.get('token');

        if (!token) {
            show(elNoToken);
            return;
        }

        // Token sending for verification
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

    // ---------- Logout dialog ----------
    (function initLogoutDialog() {
        // Dialog element definition
        const trigger = document.getElementById('logoutTrigger');
        const dialog = document.getElementById('logoutDialog');
        if (!trigger || !dialog) return;

        const confirmBtn = document.getElementById('logoutConfirm');

        // Open
        trigger.addEventListener('click', (e) => {
            e.preventDefault();
            dialog.showModal();
        });

        // Cancel + backdrop
        wireDialogClose(dialog);

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

    // ---------- Magic link verifying page ----------
    (function initMagicLogin() {
        // Element definition
        const elLoading = document.getElementById('magic-loading');
        if (!elLoading) return;

        const elSuccess = document.getElementById('magic-success');
        const elError = document.getElementById('magic-error');
        const elNoToken = document.getElementById('magic-no-token');

        function show(el) {
            [elLoading, elSuccess, elError, elNoToken].forEach(x => x && x.classList.add('d-none'));
            el.classList.remove('d-none');
        }

        // Token definition
        const token = new URLSearchParams(window.location.search).get('token');

        if (!token) {
            show(elNoToken);
            return;
        }

        // Send token
        fetch('/api/auth/magic-link/verify', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token })
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
                    // Store token
                    localStorage.setItem('user', JSON.stringify(data.user));
                }

                // Succes dialog + redirect to dashboard with user data
                show(elSuccess);
                setTimeout(() => { window.location.replace('dashboard.html'); }, 300);
            })
            .catch(() => {
                document.getElementById('magic-error-message').textContent =
                    'Network error. Please check your connection and try again.';
                show(elError);
            });
    })();

    // ---------- Password change dialog ----------
    (function initChangePasswordDialog() {
        // Element definition
        const dialog = document.getElementById('changePasswordDialog');
        if (!dialog) return;

        const email = document.getElementById('cpEmail');
        const current = document.getElementById('cpCurrentPassword');
        const newPw = document.getElementById('cpNewPassword');
        const confirm = document.getElementById('cpConfirmPassword');
        const ruleList = document.getElementById('cpRuleList');
        const strengthFill = document.getElementById('cpStrengthFill');
        const strengthLabel = document.getElementById('cpStrengthLabel');
        const emailFeedback = document.getElementById('cpEmailFeedback');
        const alertEl = document.getElementById('changePwAlert');
        const submitBtn = document.getElementById('cpSubmitBtn');

        // Password change alert
        function showAlert(message, type) { setAlert(alertEl, message, type); }
        function hideAlert() { clearAlert(alertEl); }

        // Email validation
        function validateEmail() {
            const msg = emailError(email.value);
            emailFeedback.textContent = msg;
            return mark(email, msg === '');
        }

        // Password validation
        function validateCurrent() {
            return mark(current, current.value.length > 0);
        }

        function validateNewPw() { return mark(newPw, passwordMeetsAll(newPw.value)); }
        function validateConfirm() { return mark(confirm, confirm.value.length > 0 && confirm.value === newPw.value); }

        // Password progress bar
        function updatePasswordUI() {
            renderStrength(ruleList, strengthFill, strengthLabel, newPw.value);
            if (confirm.value.length > 0) validateConfirm();
        }

        const bindLive = createLiveBinder();

        // Validation on every change
        bindLive(email, validateEmail);
        bindLive(current, validateCurrent);
        bindLive(newPw, validateNewPw);
        bindLive(confirm, validateConfirm);

        // Password bar update on change
        newPw.addEventListener('input', updatePasswordUI);
        newPw.addEventListener('blur', updatePasswordUI);

        // Cancel + backdrop
        wireDialogClose(dialog);

        // Password show/hide toggles
        bindPasswordToggle('cpPwToggle1', current);
        bindPasswordToggle('cpPwToggle2', newPw);
        bindPasswordToggle('cpPwToggle3', confirm);

        // Reset state every time the dialog opens (email comes pre-filled from dialog 1)
        dialog.addEventListener('close', () => {
            hideAlert();
        });

        updatePasswordUI();

        // Submit
        submitBtn.addEventListener('click', async () => {
            hideAlert();

            // Validation check
            const results = [validateEmail(), validateCurrent(), validateNewPw(), validateConfirm()];
            updatePasswordUI();
            if (results.includes(false)) {
                const firstInvalid = dialog.querySelector('.is-invalid');
                if (firstInvalid) firstInvalid.focus();
                return;
            }

            const originalText = submitBtn.innerHTML;
            submitBtn.disabled = true;
            submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Changing…';

            // Send data
            try {
                const res = await fetch('/api/auth/password-reset', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        email: email.value.trim(),
                        currentPassword: current.value,
                        newPassword: newPw.value
                    })
                });
                const data = await res.json().catch(() => ({}));

                email.value = '';
                current.value = '';
                newPw.value = '';
                confirm.value = '';

                // Reset validation state
                [email, current, newPw, confirm].forEach(el => {
                    el.classList.remove('is-valid', 'is-invalid');
                });

                // Reset strength meter
                strengthFill.style.width = '0%';
                strengthFill.style.background = '#e2e8f0';
                strengthLabel.textContent = 'Password strength';
                ruleList.querySelectorAll('li').forEach(li => {
                    li.classList.remove('ok');
                    li.querySelector('i').className = 'bi bi-circle';
                });

                showAlert(
                    'If an account exists with those credentials and the current password has expired, the new password has been set. Try signing in with it.',
                    'info'
                );


            } catch {
                showAlert('Network error. Please check your connection and try again.', 'danger');
            } finally {
                submitBtn.disabled = false;
                submitBtn.innerHTML = originalText;
            }
        });
    })();

    // ---------- 2FA page ----------
    (function initTwoFactorPage() {
        // Element definition
        const form = document.getElementById('twoFactorForm');
        if (!form) return;

        const codeInput = document.getElementById('tfCode');
        const targetEmail = document.getElementById('tfTargetEmail');
        const alertEl = document.getElementById('tfAlert');
        const backLink = document.getElementById('tfBackLink');

        const pendingToken = sessionStorage.getItem('pending_token');
        const pendingEmail = sessionStorage.getItem('pending_email');

        // Token check
        if (!pendingToken) {
            window.location.replace('login.html');
            return;
        }

        if (pendingEmail && targetEmail) {
            targetEmail.textContent = pendingEmail;
        }

        // 2FA alert
        function showAlert(message, type) {
            setAlert(alertEl, message, type);
            alertEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }
        function hideAlert() { clearAlert(alertEl); }

        // Code validation
        function validateCode() {
            const v = codeInput.value.trim();
            return mark(codeInput, /^\d{6}$/.test(v));
        }

        codeInput.addEventListener('input', () => {
            codeInput.value = codeInput.value.replace(/\D/g, '').slice(0, 6);
            if (codeInput.classList.contains('is-invalid')) validateCode();
        });
        codeInput.addEventListener('blur', validateCode);

        // Use different account
        backLink.addEventListener('click', (e) => {
            e.preventDefault();
            sessionStorage.removeItem('pending_token');
            sessionStorage.removeItem('pending_email');
            window.location.href = 'login.html';
        });

        setTimeout(() => codeInput.focus(), 100);

        // Submit
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideAlert();
            if (!validateCode()) { codeInput.focus(); return; }

            const submitBtn = form.querySelector('button[type="submit"]');
            const originalText = submitBtn.textContent;
            submitBtn.disabled = true;
            submitBtn.textContent = 'Verifying…';

            // Send code
            try {
                const res = await fetch('/api/twofactor/verify', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        pendingToken: pendingToken,
                        code: codeInput.value.trim()
                    })
                });

                // Response check
                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    const raw = (data.message || '').toLowerCase();
                    const isSessionExpired = res.status === 404 || data.code === 5 ||
                        raw.includes('session expired') ||
                        raw.includes('login session');
                    const isTooManyAttempts = raw.includes('too many');
                    const isCodeExpired = raw.includes('code expired') ||
                        raw.includes('not found');
                    const isWrongCode = raw.includes('do not match') ||
                        raw.includes('invalid verification code');

                    if (isSessionExpired) {
                        sessionStorage.removeItem('pending_token');
                        sessionStorage.removeItem('pending_email');
                        showAlert('Your login session expired. Redirecting to sign in…', 'warning');
                        setTimeout(() => { window.location.href = 'login.html'; }, 2000);

                    } else if (isTooManyAttempts) {
                        sessionStorage.removeItem('pending_token');
                        sessionStorage.removeItem('pending_email');
                        showAlert('Too many incorrect attempts. Redirecting to sign in for a new code…', 'warning');
                        setTimeout(() => { window.location.href = 'login.html'; }, 2500);

                    } else if (isCodeExpired) {
                        sessionStorage.removeItem('pending_token');
                        sessionStorage.removeItem('pending_email');
                        showAlert('Your code has expired. Please sign in again for a new code.', 'warning');
                        setTimeout(() => { window.location.href = 'login.html'; }, 2000);

                    } else if (isWrongCode) {
                        showAlert('Incorrect code. Check the 6 digits and try again.', 'danger');
                        codeInput.select();

                    } else if (res.status >= 500 || data.code === 13) {
                        showAlert('Server error. Please try again in a moment.', 'danger');

                    } else {
                        showAlert('Verification failed. Please try again.', 'danger');
                    }
                    return;
                }

                if (!data.token) {
                    showAlert('Unexpected server response. Please try again.', 'danger');
                    return;
                }

                // Decode JWT
                const claims = decodeJwtPayload(data.token);
                const userObj = {
                    id: claims.sub || '',
                    email: claims.email || pendingEmail || '',
                    firstName: '',
                    lastName: '',
                    passwordValid: null
                };

                // Set token and user
                localStorage.setItem('token', data.token);
                localStorage.setItem('user', JSON.stringify(userObj));

                // Remove pending data
                sessionStorage.removeItem('pending_token');
                sessionStorage.removeItem('pending_email');

                // Succes message + redirect
                showAlert('Signed in. Redirecting…', 'success');
                setTimeout(() => { window.location.href = 'dashboard.html'; }, 800);

            } catch {
                showAlert('Network error. Please check your connection and try again.', 'danger');
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = originalText;
            }
        });
    })();

    // ---------- Folder creating dialog----------
    (function initCreateFolderDialog() {
        // Element definition
        const trigger = document.getElementById('newFolderButton');
        const dialog = document.getElementById('createFolderDialog');
        if (!trigger || !dialog) return;

        const input = document.getElementById('folderNameInput');
        const confirmBtn = document.getElementById('createFolderConfirm');
        const alertEl = document.getElementById('folderAlert');

        trigger.addEventListener('click', (e) => {
            e.preventDefault();
            input.value = '';
            alertEl.classList.add('d-none');
            dialog.showModal();
            setTimeout(() => input.focus(), 100);
        });

        // Cancel + backdrop
        wireDialogClose(dialog);

        confirmBtn.addEventListener('click', async () => {
            const name = input.value.trim();

            // Basic name validation
            if (!name) {
                showFolderError('Please enter a folder name.');
                return;
            }
            if (name.includes('/') || name.includes('\\')) {
                showFolderError('Folder name cannot contain slashes.');
                return;
            }
            if (name.length > 100) {
                showFolderError('Folder name is too long (max 100 characters).');
                return;
            }

            // Build the path relative to the current folder
            const parent = currentFolderPath.endsWith('/') || currentFolderPath === ''
                ? currentFolderPath
                : currentFolderPath + '/';
            const newPath = parent + name;

            const originalText = confirmBtn.innerHTML;
            confirmBtn.disabled = true;
            confirmBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Creating…';
            alertEl.classList.add('d-none');

            try {
                const token = localStorage.getItem('token');
                const res = await fetch('/api/storage/folder', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({
                        path: newPath,
                        token: token
                    })
                });

                if (!res.ok) {
                    const data = await res.json().catch(() => ({}));
                    throw new Error(data.message || `Failed to create folder (HTTP ${res.status})`);
                }

                dialog.close('success');

                // Reload the *current* folder, not the page
                window.dispatchEvent(new CustomEvent('files-changed'));
                // Or, if load() is reachable:
                // load(currentFolderPath);

            } catch (err) {
                alertEl.textContent = err.message;
                alertEl.className = 'alert alert-danger mb-3';
                alertEl.classList.remove('d-none');
            } finally {
                confirmBtn.disabled = false;
                confirmBtn.innerHTML = originalText;
            }
        });

        function showFolderError(msg) {
            alertEl.textContent = msg;
            alertEl.className = 'alert alert-danger mb-3';
            alertEl.classList.remove('d-none');
        }
    })();

    // ---------- File uploading dialog----------
    (function initUploadFileDialog() {
        // Element definition
        const dialog = document.getElementById('uploadFileDialog');
        if (!dialog) return;

        const triggerNavbar = document.getElementById('uploadButtonNavbar');
        const triggerPage = document.getElementById('uploadButtonPage');
        const fileInput = document.getElementById('uploadFileInput');
        const alertEl = document.getElementById('uploadAlert');
        const fileInfo = document.getElementById('uploadFileInfo');
        const fileNameEl = document.getElementById('uploadFileName');
        const fileSizeEl = document.getElementById('uploadFileSize');
        const progressWrap = document.getElementById('uploadProgressWrap');
        const progressBar = document.getElementById('uploadProgressBar');
        const progressText = document.getElementById('uploadProgressText');
        const submitBtn = document.getElementById('uploadFileConfirm');

        const MAX_SIZE = 100 * 1024 * 1024; // 100 MB

        function formatSize(bytes) {
            if (bytes < 1024) return bytes + ' B';
            if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
            if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
            return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
        }

        // upload alert
        function showAlert(message, type) { setAlert(alertEl, message, type); }
        function hideAlert() { clearAlert(alertEl); }

        // Dialog reset
        function resetDialog() {
            hideAlert();
            fileInput.value = '';
            fileInput.classList.remove('is-valid', 'is-invalid');
            fileInfo.classList.add('d-none');
            progressWrap.classList.add('d-none');
            progressBar.style.width = '0%';
            progressBar.classList.remove('bg-danger');
            progressText.textContent = 'Uploading… 0%';
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<i class="bi bi-cloud-arrow-up me-1"></i> Upload';
        }

        // Open dialog
        function openDialog(e) {
            if (e) e.preventDefault();
            resetDialog();
            dialog.showModal();
            setTimeout(() => fileInput.focus(), 100);
        }

        // Opening button click
        if (triggerNavbar) triggerNavbar.addEventListener('click', openDialog);
        if (triggerPage) triggerPage.addEventListener('click', openDialog);

        // Cancel + backdrop
        wireDialogClose(dialog);

        // File validation
        fileInput.addEventListener('change', () => {
            hideAlert();
            const file = fileInput.files && fileInput.files[0];
            if (!file) {
                fileInfo.classList.add('d-none');
                fileInput.classList.remove('is-valid', 'is-invalid');
                return;
            }
            if (file.size > MAX_SIZE) {
                fileInput.classList.add('is-invalid');
                fileInput.classList.remove('is-valid');
                fileInfo.classList.add('d-none');
                showAlert(`File is too large (${formatSize(file.size)}). Maximum allowed size is ${formatSize(MAX_SIZE)}.`, 'danger');
                return;
            }
            fileInput.classList.remove('is-invalid');
            fileInput.classList.add('is-valid');
            fileNameEl.textContent = file.name;
            fileSizeEl.textContent = formatSize(file.size);
            fileInfo.classList.remove('d-none');
        });

        // Upload
        submitBtn.addEventListener('click', async () => {
            hideAlert();

            // Validation
            const file = fileInput.files && fileInput.files[0];
            if (!file) {
                fileInput.classList.add('is-invalid');
                showAlert('Please select a file to upload.', 'danger');
                return;
            }
            if (file.size > MAX_SIZE) {
                showAlert(`File is too large (${formatSize(file.size)}). Maximum allowed size is ${formatSize(MAX_SIZE)}.`, 'danger');
                return;
            }

            const originalHtml = submitBtn.innerHTML;
            submitBtn.disabled = true;
            submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Uploading…';

            const formData = new FormData();
            formData.append('file', file);

            // Current folder path
            formData.append('path', currentFolderPath);

            const token = localStorage.getItem('token');

            // Send file
            try {
                const res = await fetch('/api/storage/file', {
                    method: 'POST',
                    headers: token ? { 'Authorization': `Bearer ${token}` } : {},
                    body: formData
                });

                // Response check
                if (!res.ok) {
                    if (res.status === 401) {
                        showAlert('Your session has expired. Redirecting to sign in…', 'danger');
                        setTimeout(() => { window.location.href = 'login.html'; }, 2000);
                        return;
                    }
                    if (res.status === 400) {
                        showAlert('Invalid upload request. Please select a file and try again.', 'danger');
                        return;
                    }
                    if (res.status === 413) {
                        showAlert('File is too large for the server. Try a smaller file.', 'danger');
                        return;
                    }
                    if (res.status === 502) {
                        showAlert('Storage service is temporarily unavailable. Please try again in a moment.', 'danger');
                        return;
                    }
                    if (res.status >= 500) {
                        showAlert('Server error while uploading. Please try again in a moment.', 'danger');
                        return;
                    }
                    showAlert('Upload failed. Please try again.', 'danger');
                    return;
                }

                // Success message + close
                showAlert('File uploaded successfully.', 'success');
                setTimeout(() => {
                    dialog.close('uploaded');
                    window.dispatchEvent(new CustomEvent('files-changed'));
                }, 1200);

            } catch {
                showAlert('Network error during upload. Please check your connection and try again.', 'danger');
            } finally {
                submitBtn.disabled = false;
                submitBtn.innerHTML = originalHtml;
            }
        });
    })();

    // ---------- File list and folder browsing ----------
    (function initFileList() {
        'use strict';

        // Element definition
        const tbody = document.getElementById('fileTableBody');
        if (!tbody) return;

        // Breadcrumb and item-count elements
        const bc = document.querySelector('.breadcrumb');
        const fileCountEl = document.getElementById('fileCount');

        // Build a URL for a folder path - real href so ctrl+click / open-in-new-tab work
        function buildNavUrl(path) {
            const url = new URL(window.location.href);
            if (path) url.searchParams.set('path', path);
            else url.searchParams.delete('path');
            return url.pathname + url.search;
        }

        // Read ?path= from the current URL (for deep-linking)
        function pathFromUrl() {
            return new URLSearchParams(window.location.search).get('path') || '';
        }

        // Breadcrumb
        function renderBreadcrumb(dirPath) {
            if (!bc) return;
            bc.innerHTML = '';

            const root = document.createElement('li');
            root.className = 'breadcrumb-item' + (dirPath ? '' : ' active');
            if (dirPath) {
                // Clickable root - returns to My Drive
                const a = document.createElement('a');
                a.href = buildNavUrl('');
                a.textContent = 'My Drive';
                a.dataset.navPath = '';
                root.appendChild(a);
            } else {
                // At root - no link
                root.textContent = 'My Drive';
                root.setAttribute('aria-current', 'page');
            }
            bc.appendChild(root);

            // Path segments - intermediate ones clickable, last one is current
            const parts = dirPath.split('/').filter(Boolean);
            parts.forEach((part, i) => {
                const isLast = i === parts.length - 1;
                const li = document.createElement('li');
                li.className = 'breadcrumb-item' + (isLast ? ' active' : '');
                if (isLast) {
                    li.textContent = part;   // textContent - never innerHTML for paths
                    li.setAttribute('aria-current', 'page');
                } else {
                    // Build cumulative path for navigation
                    const target = parts.slice(0, i + 1).join('/') + '/';
                    const a = document.createElement('a');
                    a.href = buildNavUrl(target);
                    a.textContent = part;
                    a.dataset.navPath = target;
                    li.appendChild(a);
                }
                bc.appendChild(li);
            });
        }

        // Delegated breadcrumb click - survives re-renders
        if (bc) {
            bc.addEventListener('click', (e) => {
                const a = e.target.closest('a[data-nav-path]');
                if (!a) return;
                // Modifier keys - let the browser open in new tab
                if (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) return;
                e.preventDefault();
                load(a.dataset.navPath);
            });
        }

        // File size formatter
        function formatSize(bytes) {
            if (!bytes || bytes <= 0) return '—';
            if (bytes < 1024) return bytes + ' B';
            if (bytes < 1024 ** 2) return (bytes / 1024).toFixed(1) + ' KB';
            if (bytes < 1024 ** 3) return (bytes / 1024 ** 2).toFixed(1) + ' MB';
            return (bytes / 1024 ** 3).toFixed(2) + ' GB';
        }

        // Single row
        function row(file) {
            const icon = file.isDirectory ? 'bi-folder-fill' : 'bi-file-earmark-fill';
            const kind = file.isDirectory ? 'Folder' : 'File';

            const tr = document.createElement('tr');
            tr.innerHTML = `
        <td>
            <div class="file-name">
                <span class="icon ${file.isDirectory ? 'folder' : 'doc'}">
                    <i class="bi ${icon}"></i>
                </span>
                <div>${escapeHtml(file.name)}<small>${kind}</small></div>
            </div>
        </td>
        <td class="d-none d-md-table-cell text-soft">${formatSize(file.size)}</td>
        <td class="d-none d-md-table-cell text-soft">—</td>
        <td class="d-none d-sm-table-cell"></td>
        <td class="file-actions">
            <button type="button" class="btn-icon" aria-label="More options">
                <i class="bi bi-three-dots"></i>
            </button>
        </td>`;

            // Folder row - click navigates into folder
            if (file.isDirectory) {
                tr.style.cursor = 'pointer';
                tr.addEventListener('click', (e) => {
                    if (e.target.closest('button')) return;
                    load(file.path);
                });
            }

            // Kebab menu opens the rename dialog (folders only)
            const menuBtn = tr.querySelector('.file-actions .btn-icon');
            if (menuBtn) {
                menuBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    if (file.isDirectory) {
                        if (window.openRenameFolderDialog) window.openRenameFolderDialog(file);
                    } else {
                        if (window.openRenameFileDialog) window.openRenameFileDialog(file);
                    }
                });
            }

            return tr;
        }

        // Escape HTML to prevent XSS from file/folder names
        function escapeHtml(s) {
            return s.replace(/[&<>"']/g, c => ({
                '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
            }[c]));
        }

        // Load folder contents
        async function load(dirPath, push = true) {
            const token = localStorage.getItem('token');
            if (!token) { window.location.replace('login.html'); return; }

            // Loading state
            tbody.innerHTML = `<tr><td colspan="5" class="text-soft p-3">Loading…</td></tr>`;

            // Token in query - gateway drops Authorization for GET
            const qs = new URLSearchParams();
            if (dirPath) qs.set('path', dirPath);
            qs.set('token', token);

            try {
                const res = await fetch('/api/storage/files?' + qs.toString());
                if (res.status === 401) { window.location.replace('login.html'); return; }
                if (!res.ok) {
                    const data = await res.json().catch(() => ({}));
                    throw new Error(data.message || `HTTP ${res.status}`);
                }

                const data = await res.json();
                const files = data.files || [];

                // Item count in the header
                if (fileCountEl) {
                    fileCountEl.textContent = files.length === 0
                        ? 'No items'
                        : files.length + (files.length === 1 ? ' item' : ' items');
                }

                // Commit path + breadcrumb only after a successful load
                currentFolderPath = dirPath;
                renderBreadcrumb(dirPath);

                // Push URL state - skip when driven by popstate or refresh
                if (push) history.pushState({ path: dirPath }, '', buildNavUrl(dirPath));

                tbody.innerHTML = '';
                // Empty folder message
                if (files.length === 0) {
                    tbody.innerHTML = `<tr><td colspan="5" class="text-soft p-3">This folder is empty.</td></tr>`;
                    return;
                }

                // Folders first, then files, alphabetical
                files
                    .slice()
                    .sort((a, b) => (b.isDirectory - a.isDirectory) || a.name.localeCompare(b.name))
                    .forEach(f => tbody.appendChild(row(f)));

            } catch (err) {
                // Error state
                if (fileCountEl) fileCountEl.textContent = '—';
                tbody.innerHTML =
                    `<tr><td colspan="5" class="text-danger p-3">Failed to load files: ${escapeHtml(err.message)}</td></tr>`;
            }
        }

        // Browser back/forward
        window.addEventListener('popstate', () => load(pathFromUrl(), false));

        // Refresh on change
        window.addEventListener('files-changed', () => load(currentFolderPath, false));

        // Initial load
        load(pathFromUrl(), false);
    })();

    // ---------- Folder renaming dialog ----------
    let openRenameDialog = null;

    (function initFolderRename() {
        // Element definition
        const dialog = document.getElementById('renameFolderDialog');
        if (!dialog) return;

        const input = document.getElementById('renameFolderInput');
        const confirmBtn = document.getElementById('renameFolderConfirm');
        const alertEl = document.getElementById('renameFolderAlert');

        let target = null;

        // Dialog alert
        function showError(msg) {
            alertEl.textContent = msg;
            alertEl.className = 'alert alert-danger mb-3';
            alertEl.classList.remove('d-none');
        }

        // Cancel + backdrop
        wireDialogClose(dialog);

        // Called from the folder row kebab
        openRenameDialog = function (file) {
            // Split file.path into parent + leaf for newPath construction
            const trimmed = file.path.endsWith('/') ? file.path.slice(0, -1) : file.path;
            const lastSlash = trimmed.lastIndexOf('/');
            const parent = lastSlash >= 0 ? trimmed.slice(0, lastSlash + 1) : '';

            target = { oldPath: file.path, parent, leafName: file.name };

            input.value = file.name;
            alertEl.classList.add('d-none');
            dialog.showModal();
            setTimeout(() => { input.focus(); input.select(); }, 100);
        };

        // Enter submits
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') { e.preventDefault(); confirmBtn.click(); }
        });

        // Confirm button
        confirmBtn.addEventListener('click', async () => {
            if (!target) return;

            const newName = input.value.trim();

            // Name validation
            if (!newName) { showError('Please enter a name.'); return; }
            if (newName === target.leafName) { dialog.close('cancel'); return; }
            if (newName.includes('/') || newName.includes('\\')) { showError('Name cannot contain slashes.'); return; }
            if (newName.length > 100) { showError('Name is too long (max 100 characters).'); return; }

            const token = localStorage.getItem('token');
            if (!token) { window.location.replace('login.html'); return; }

            // Build new path - folders always end with a slash
            const newPath = target.parent + newName + '/';

            // Loading state
            const original = confirmBtn.innerHTML;
            confirmBtn.disabled = true;
            confirmBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Renaming…';
            alertEl.classList.add('d-none');

            // Send rename request
            try {
                const res = await fetch('/api/storage/folder', {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({
                        oldPath: target.oldPath,
                        newPath: newPath,
                        token: token
                    })
                });

                // Response check
                const data = await res.json().catch(() => ({}));
                if (!res.ok) throw new Error(data.message || `Rename failed (HTTP ${res.status})`);

                // Success - close and refresh table
                dialog.close('success');
                window.dispatchEvent(new CustomEvent('files-changed'));

            } catch (err) {
                showError(err.message);
            } finally {
                // Restore button
                confirmBtn.disabled = false;
                confirmBtn.innerHTML = original;
            }
        });
    })();

    // ---------- File renaming dialog ----------
    window.openRenameFileDialog = null;

    (function initFileRename() {
        // Element definition
        const dialog = document.getElementById('renameFileDialog');
        if (!dialog) return;

        const input = document.getElementById('renameFileInput');
        const confirmBtn = document.getElementById('renameFileConfirm');
        const alertEl = document.getElementById('renameFileAlert');

        let target = null;

        // Dialog alert
        function showError(msg) {
            alertEl.textContent = msg;
            alertEl.className = 'alert alert-danger mb-3';
            alertEl.classList.remove('d-none');
        }

        // Cancel + backdrop
        wireDialogClose(dialog);

        // Called from the file row kebab
        window.openRenameFileDialog = function (file) {
            // We only need the relative path for the backend
            target = { oldPath: file.path, leafName: file.name };

            input.value = file.name;
            alertEl.classList.add('d-none');
            dialog.showModal();
            setTimeout(() => { input.focus(); input.select(); }, 100);
        };

        // Enter submits
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') { e.preventDefault(); confirmBtn.click(); }
        });

        // Confirm button
        confirmBtn.addEventListener('click', async () => {
            if (!target) return;

            const newName = input.value.trim();

            // Name validation
            if (!newName) { showError('Please enter a name.'); return; }
            if (newName === target.leafName) { dialog.close('cancel'); return; }
            if (newName.includes('/') || newName.includes('\\')) { showError('Name cannot contain slashes.'); return; }
            if (newName.length > 100) { showError('Name is too long (max 100 characters).'); return; }

            const token = localStorage.getItem('token');
            if (!token) { window.location.replace('login.html'); return; }

            // Loading state
            const original = confirmBtn.innerHTML;
            confirmBtn.disabled = true;
            confirmBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Renaming…';
            alertEl.classList.add('d-none');

            // Send rename request
            try {
                const res = await fetch('/api/storage/files', {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({
                        old_path: target.oldPath,
                        new_name: newName,
                        token: token
                    })
                });

                // Response check
                const data = await res.json().catch(() => ({}));
                if (!res.ok) throw new Error(data.message || `Rename failed (HTTP ${res.status})`);

                // Success - close and refresh table
                dialog.close('success');
                window.dispatchEvent(new CustomEvent('files-changed'));

            } catch (err) {
                showError(err.message);
            } finally {
                // Restore button
                confirmBtn.disabled = false;
                confirmBtn.innerHTML = original;
            }
        });
    })();

    // ---------- Notifications page ----------
    (function initNotificationsPage() {
        'use strict';

        // Element definition
        const container = document.getElementById('notificationsContainer');
        if (!container) return;

        const countEl = document.getElementById('notificationCount');

        // Formatting
        function timeAgo(iso) {
            let t;
            if (iso && typeof iso === 'object' && iso.seconds != null) {
                t = Number(iso.seconds) * 1000;
            } else {
                t = new Date(iso).getTime();
            }
            if (!Number.isFinite(t)) return '';

            const diff = Math.max(0, Date.now() - t);
            const sec = Math.floor(diff / 1000);
            const min = Math.floor(sec / 60);
            const hr = Math.floor(min / 60);
            const day = Math.floor(hr / 24);

            if (sec < 60) return 'just now';
            if (min < 60) return min + (min === 1 ? ' minute ago' : ' minutes ago');
            if (hr < 24) return hr + (hr === 1 ? ' hour ago' : ' hours ago');
            if (day < 7) return day + (day === 1 ? ' day ago' : ' days ago');
            return new Date(t).toLocaleDateString();
        }

        function groupByDay(items) {
            const now = new Date();
            const startToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
            const startYesterday = startToday - 86400000;

            const groups = { today: [], yesterday: [], earlier: [] };
            for (const n of items) {
                const raw = n.createdAt;
                const t = (raw && typeof raw === 'object' && raw.seconds != null)
                    ? Number(raw.seconds) * 1000
                    : new Date(raw).getTime();
                if (t >= startToday) groups.today.push(n);
                else if (t >= startYesterday) groups.yesterday.push(n);
                else groups.earlier.push(n);
            }
            return groups;
        }

        function iconFor(type) {
            switch (type) {
                case 'share': return 'bi-people-fill';
                case 'new_version': return 'bi-arrow-up-circle-fill';
                case 'new_file': return 'bi-file-earmark-plus-fill';
                default: return 'bi-info-circle-fill';
            }
        }

        function escapeHtml(s) {
            return String(s || '').replace(/[&<>"']/g, c => ({
                '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
            }[c]));
        }

        // Adding items as a block
        function renderItem(n) {
            const iconType = ['share', 'new_version', 'new_file', 'system'].includes(n.type)
                ? n.type
                : 'system';

            const item = document.createElement('div');
            item.className = 'notification-item' + (n.isRead ? '' : ' unread');

            item.innerHTML = `
            <div class="notification-icon type-${iconType}">
                <i class="bi ${iconFor(n.type)}"></i>
            </div>
            <div class="notification-content">
                <div class="notification-title">${escapeHtml(n.title)}</div>
                <div class="notification-body">${escapeHtml(n.body)}</div>
                <div class="notification-time">${escapeHtml(timeAgo(n.createdAt))}</div>
            </div>
            <span class="notification-dot"></span>`;

            if (n.resourceType === 'folder' && n.resourceId) {
                item.style.cursor = 'pointer';
                item.addEventListener('click', () => {
                    window.location.href = 'dashboard.html?path=' + encodeURIComponent(n.resourceId);
                });
            }

            return item;
        }

        function renderGroup(label, items) {
            if (items.length === 0) return null;
            const section = document.createElement('div');
            section.className = 'notification-group';
            section.innerHTML = `<div class="notification-group-label">${escapeHtml(label)}</div>`;
            for (const n of items) section.appendChild(renderItem(n));
            return section;
        }

        // Load
        async function load() {
            const token = localStorage.getItem('token');
            if (!token) { window.location.replace('login.html'); return; }

            container.innerHTML = '<div class="notification-empty">Loading notifications…</div>';

            try {
                // Send the JWT
                const res = await fetch('/api/notifications', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({ token: token })
                });

                if (res.status === 401) {
                    // Token is invalid or expired - kick back to login
                    localStorage.removeItem('token');
                    localStorage.removeItem('user');
                    window.location.replace('login.html');
                    return;
                }

                if (!res.ok) {
                    const data = await res.json().catch(() => ({}));
                    throw new Error(data.message || `HTTP ${res.status}`);
                }

                const data = await res.json();
                const notifications = data.notifications || [];

                if (countEl) {
                    if (notifications.length === 0) {
                        countEl.textContent = 'No notifications';
                    } else {
                        const unread = notifications.filter(n => !n.isRead).length;
                        countEl.textContent = unread > 0
                            ? unread + ' unread'
                            : notifications.length + (notifications.length === 1 ? ' notification' : ' notifications');
                    }
                }

                // No notification
                if (notifications.length === 0) {
                    container.innerHTML = `
                    <div class="notification-empty">
                        <i class="bi bi-bell-slash" style="font-size:2.5rem; color:#c7cbd6;"></i>
                        <div class="mt-2">No notifications yet.</div>
                        <div style="font-size:0.82rem; margin-top:0.4rem;">When someone shares a folder or file with you, you'll see it here.</div>
                    </div>`;
                    return;
                }

                // Sort
                notifications.sort((a, b) => {
                    const ta = (a.createdAt && typeof a.createdAt === 'object' && a.createdAt.seconds != null)
                        ? Number(a.createdAt.seconds) : new Date(a.createdAt).getTime() / 1000;
                    const tb = (b.createdAt && typeof b.createdAt === 'object' && b.createdAt.seconds != null)
                        ? Number(b.createdAt.seconds) : new Date(b.createdAt).getTime() / 1000;
                    return tb - ta;
                });

                const groups = groupByDay(notifications);

                container.innerHTML = '';
                const sections = [
                    renderGroup('Today', groups.today),
                    renderGroup('Yesterday', groups.yesterday),
                    renderGroup('Earlier', groups.earlier),
                ];
                for (const s of sections) if (s) container.appendChild(s);

            } catch (err) {
                container.innerHTML =
                    `<div class="notification-empty text-danger">Failed to load notifications: ${escapeHtml(err.message)}</div>`;
            }
        }

        // Load
        load();
    })();

    // ---------- Admin Dashboard ----------
    (function initAdminDashboard() {
        'use strict';

        const requestsTbody = document.getElementById('quotaRequestTableBody');
        const usersTbody = document.getElementById('allUsersTableBody');
        if (!requestsTbody || !usersTbody) return;

        const token = localStorage.getItem('token');
        if (!token) { window.location.replace('login.html'); return; }

        // Universal size formatter (Input is in Megabytes)
        function formatSize(mb) {
            if (!mb || mb <= 0) return '0 MB';
            if (mb < 1024) return mb + ' MB';
            return (mb / 1024).toFixed(2) + ' GB';
        }

        // Load pending requests
        async function loadRequests() {
            try {
                const res = await fetch('/api/quota/requests', {
                    headers: { 'Authorization': 'Bearer ' + token }
                });

                if (res.status === 401) { window.location.replace('login.html'); return; }
                if (!res.ok) throw new Error(`HTTP ${res.status}`);

                const data = await res.json();
                const requests = data.requests || [];


                // Update badge
                const badge = document.getElementById('pendingBadge');
                if (badge) badge.textContent = requests.length;

                if (requests.length === 0) {
                    requestsTbody.innerHTML = `<tr><td colspan="5" class="text-soft p-3 text-center">No pending requests.</td></tr>`;
                    return;
                }

                requestsTbody.innerHTML = '';
                requests.forEach(req => {
                    // Convert string numbers to numbers
                    const currentQuota = Number(req.storageQuota || 0);
                    const usedQuota = Number(req.usedQuota || 0);
                    const requestedAmt = Number(req.amount || 0);

                    // Fallbacks for missing names
                    const fName = req.firstName || '';
                    const lName = req.lastName || '';
                    const email = req.email || 'Unknown Email';
                    const fullName = (fName || lName) ? `${fName} ${lName}`.trim() : 'Unknown User';

                    const tr = document.createElement('tr');
                    tr.innerHTML = `
                        <td>
                            <div class="fw-semibold">${fullName}</div>
                            <small class="text-soft">${email}</small>
                        </td>
                        <td class="text-soft">${formatSize(currentQuota)}</td>
                        <td class="text-soft">${formatSize(usedQuota)}</td>
                        <td class="fw-semibold text-success">+${formatSize(requestedAmt)}</td>
                        <td class="file-actions text-end">
                            <button class="btn btn-sm btn-success rounded-pill px-3 me-1" onclick="handleQuotaAction('${email}', true)">Approve</button>
                            <button class="btn btn-sm btn-outline-danger rounded-pill px-3" onclick="handleQuotaAction('${email}', false)">Reject</button>
                        </td>
                    `;
                    requestsTbody.appendChild(tr);
                });
            } catch (err) {
                requestsTbody.innerHTML = `<tr><td colspan="5" class="text-danger p-3 text-center">Failed to load requests: ${err.message}</td></tr>`;
            }
        }

        // Load users
        async function loadUsers() {
            try {
                const res = await fetch('/api/users', {
                    headers: { 'Authorization': 'Bearer ' + token }
                });

                if (res.status === 401) { window.location.replace('login.html'); return; }
                if (!res.ok) throw new Error(`HTTP ${res.status} - Endpoint might not be implemented yet`);

                const data = await res.json();
                const users = data.users || [];

                console.log("API Response Data (Users):", users);

                if (users.length === 0) {
                    usersTbody.innerHTML = `<tr><td colspan="5" class="text-soft p-3 text-center">No users found.</td></tr>`;
                    return;
                }

                usersTbody.innerHTML = '';
                users.forEach(user => {
                    const used = Number(user.usedQuota || 0);
                    const total = Number(user.storageQuota || 1);
                    const percentage = Math.min(100, Math.round((used / total) * 100));

                    const fName = user.firstName || '';
                    const lName = user.lastName || '';
                    const fullName = (fName || lName) ? `${fName} ${lName}`.trim() : 'Unknown User';

                    const tr = document.createElement('tr');
                    tr.innerHTML = `
                        <td>
                            <div class="fw-semibold">${fullName}</div>
                            <small class="text-soft">ID: ${user.id || '—'}</small>
                        </td>
                        <td class="text-soft">${user.email}</td>
                        <td class="text-soft">${formatSize(total)}</td>
                        <td class="text-soft">${formatSize(used)}</td>
                        <td style="width: 20%;">
                            <div class="d-flex align-items-center gap-2">
                                <div class="progress flex-grow-1" style="height: 6px;">
                                    <div class="progress-bar ${percentage > 90 ? 'bg-danger' : 'bg-success'}" style="width: ${percentage}%"></div>
                                </div>
                                <small class="text-soft" style="width: 35px;">${percentage}%</small>
                            </div>
                        </td>
                    `;
                    usersTbody.appendChild(tr);
                });
            } catch (err) {
                usersTbody.innerHTML = `<tr><td colspan="5" class="text-danger p-3 text-center">Failed to load users: ${err.message}</td></tr>`;
            }
        }

        // ---------- Approve/Reject Action ----------
        window.handleQuotaAction = async function (email, isApprove) {
            const action = isApprove ? 'approve' : 'reject';
            if (!confirm(`Are you sure you want to ${action} the request for ${email}?`)) return;

            try {

                const res = await fetch(`/api/quota/handle`, {
                    method: 'POST',
                    headers: {
                        'Authorization': 'Bearer ' + token,
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ email: email, approve: isApprove })
                });

                if (!res.ok) {
                    const data = await res.json().catch(() => ({}));
                    throw new Error(data.message || `HTTP ${res.status}`);
                }

                // Reload
                loadRequests();

            } catch (err) {
                alert('Action failed: ' + err.message);
            }
        };


        // Load
        loadRequests();

        // Load users if tab clicked
        const usersTab = document.getElementById('users-tab');
        if (usersTab) {
            usersTab.addEventListener('click', loadUsers);
        }
    })();

    // ---------- Request Quota Dialog ----------
    (function initRequestQuotaDialog() {
        'use strict';

        // Declaration
        const triggers = document.querySelectorAll('.request-btn');
        const dialog = document.getElementById('requestQuotaDialog');
        if (!dialog || triggers.length === 0) return;

        const input = document.getElementById('quotaAmountInput');
        const confirmBtn = document.getElementById('requestQuotaConfirm');
        const alertEl = document.getElementById('quotaAlert');

        // Alert helper
        function showAlert(msg, type) {
            alertEl.textContent = msg;
            alertEl.className = `alert alert-${type} mb-3`;
            alertEl.classList.remove('d-none');
        }

        function hideAlert() {
            alertEl.classList.add('d-none');
            alertEl.textContent = '';
        }

        // Open dialog
        triggers.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                input.value = '';
                hideAlert();
                dialog.showModal();
                setTimeout(() => input.focus(), 100);
            });
        });

        // Cancel + backdrop
        wireDialogClose(dialog);

        // Submit request
        confirmBtn.addEventListener('click', async () => {
            hideAlert();
            const amount = Number(input.value);

            // Frontend validation
            if (!amount || amount <= 0) {
                showAlert('Please enter a valid amount greater than 0.', 'danger');
                return;
            }

            if (amount > 10240) {
                showAlert('You cannot request more than 10 GB (10240 MB) at once.', 'danger');
                return;
            }

            const originalText = confirmBtn.innerHTML;
            confirmBtn.disabled = true;
            confirmBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span> Sending…';

            const token = localStorage.getItem('token');

            // Send
            try {
                const res = await fetch('/api/quota/request', {
                    method: 'POST',
                    headers: {
                        'Authorization': 'Bearer ' + token,
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ amount: amount })
                });

                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    showAlert(data.message || `Request failed (HTTP ${res.status})`, 'danger');
                    return;
                }

                // Success
                showAlert('Request submitted successfully! An admin will review it soon.', 'success');
                setTimeout(() => dialog.close('success'), 2000);

            } catch (err) {
                showAlert('Network error. Please check your connection and try again.', 'danger');
            } finally {
                confirmBtn.disabled = false;
                confirmBtn.innerHTML = originalText;
            }
        });
    })();


    // ---------- Reset password page ----------
    (function initResetPasswordPage() {
        'use strict';

        // Element definition
        const form = document.getElementById('resetPasswordForm');
        if (!form) return;

        const formArea = document.getElementById('resetFormArea');
        const missingBox = document.getElementById('resetMissingToken');
        const successBox = document.getElementById('resetSuccess');
        const alertEl = document.getElementById('resetAlert');
        const submitBtn = document.getElementById('resetSubmitBtn');

        const password = document.getElementById('password');
        const confirm = document.getElementById('confirmPassword');
        const ruleList = document.getElementById('ruleList');
        const strengthFill = document.getElementById('strengthFill');
        const strengthLabel = document.getElementById('strengthLabel');

        // Token from the URL: reset-password.html?token=...
        const token = (new URLSearchParams(window.location.search).get('token') || '').trim();

        // No token — link was truncated or copied incompletely
        if (!token) {
            formArea.classList.add('d-none');
            missingBox.classList.remove('d-none');
            return;
        }

        // Alert helpers
        function showAlert(message, type) { setAlert(alertEl, message, type || 'danger'); }
        function hideAlert() { clearAlert(alertEl); }

        // Password validation — uses shared PASSWORD_RULES via passwordMeetsAll
        function validatePassword() { return mark(password, passwordMeetsAll(password.value)); }
        function validateConfirm() {
            return mark(confirm, confirm.value.length > 0 && confirm.value === password.value);
        }

        // Strength meter — shared renderStrength
        function updatePasswordUI() {
            renderStrength(ruleList, strengthFill, strengthLabel, password.value);
            if (confirm.value.length > 0) validateConfirm();
        }

        // Live validation (blur → then revalidate on input)
        const bindLive = createLiveBinder();
        bindLive(password, validatePassword);
        bindLive(confirm, validateConfirm);

        // Update the meter as the user types
        password.addEventListener('input', updatePasswordUI);
        password.addEventListener('blur', updatePasswordUI);

        // Show / hide toggles — shared bindPasswordToggle
        bindPasswordToggle('pwToggle', password);
        bindPasswordToggle('pwToggle2', confirm);

        // Initial paint
        updatePasswordUI();

        // Submit
        form.addEventListener('submit', async function (e) {
            e.preventDefault();
            hideAlert();

            // Validate
            const results = [validatePassword(), validateConfirm()];
            updatePasswordUI();
            if (results.includes(false)) {
                const firstInvalid = form.querySelector('.is-invalid');
                if (firstInvalid) {
                    firstInvalid.focus();
                    firstInvalid.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
                return;
            }

            // Guard against sending the raw token as the password
            if (password.value === token) {
                showAlert('The new password must be different from the reset token.', 'danger');
                password.focus();
                return;
            }

            const originalHtml = submitBtn.innerHTML;
            submitBtn.disabled = true;
            submitBtn.innerHTML =
                '<span class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>Updating…';

            try {
                const res = await fetch('/api/auth/forgotten-password/verify', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ reset_token: token, newPassword: password.value })
                });

                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    const raw = (data.message || '').toLowerCase();

                    if (raw.includes('expired') ||
                        raw.includes('invalid') ||
                        raw.includes('used') ||
                        res.status === 400 || res.status === 404 || res.status === 410) {
                        showAlert('This reset link is invalid or has expired. Request a new one.', 'danger');
                    } else if (res.status >= 500) {
                        showAlert('Server error. Please try again in a moment.', 'danger');
                    } else {
                        showAlert(data.message || `Reset failed (HTTP ${res.status}).`, 'danger');
                    }
                    return;
                }

                // Success — swap to the success panel
                formArea.classList.add('d-none');
                successBox.classList.remove('d-none');

            } catch {
                showAlert('Network error. Please check your connection and try again.', 'danger');
            } finally {
                submitBtn.disabled = false;
                submitBtn.innerHTML = originalHtml;
            }
        });
    })();
})();
