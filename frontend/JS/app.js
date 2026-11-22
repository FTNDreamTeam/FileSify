(function () {
    'use strict';
    let currentFolderPath = '';

    function renderBreadcrumb(dirPath) {
        const bc = document.querySelector('.breadcrumb');
        if (!bc) return;
        bc.innerHTML = '';

        const root = document.createElement('li');
        root.className = 'breadcrumb-item' + (dirPath ? '' : ' active');
        if (dirPath) {
            const a = document.createElement('a');
            a.href = '#';
            a.textContent = 'My Drive';
            a.addEventListener('click', (e) => { e.preventDefault(); load(''); });
            root.appendChild(a);
        } else {
            root.textContent = 'My Drive';
            root.setAttribute('aria-current', 'page');
        }
        bc.appendChild(root);

        const parts = dirPath.split('/').filter(Boolean);
        parts.forEach((part, i) => {
            const isLast = i === parts.length - 1;
            const li = document.createElement('li');
            li.className = 'breadcrumb-item' + (isLast ? ' active' : '');
            if (isLast) {
                li.textContent = part;            // textContent — never innerHTML for paths
                li.setAttribute('aria-current', 'page');
            } else {
                const a = document.createElement('a');
                a.href = '#';
                a.textContent = part;
                const target = parts.slice(0, i + 1).join('/') + '/';
                a.addEventListener('click', (e) => { e.preventDefault(); load(target); });
                li.appendChild(a);
            }
            bc.appendChild(li);
        });
    }

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
        length : v => v.length >= 12,
        upper  : v => /[A-Z]/.test(v),
        lower  : v => /[a-z]/.test(v),
        digit  : v => /[0-9]/.test(v),
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
            if (met <= 2)       { label = 'Weak';   color = '#dc3545'; }
            else if (met <= 3)  { label = 'Fair';   color = '#f59e0b'; }
            else if (met === 4) { label = 'Good';   color = '#0ea5e9'; }
            else                { label = 'Strong'; color = '#16a34a'; }
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
                e.clientY < r.top  || e.clientY > r.bottom;
            if (outside) dialog.close('cancel');
        });
    }

    // ---------- Login ----------
    (function initLogin() {
        // Login form definition
        const form = document.getElementById('loginForm');
        if (!form) return;

        const email         = document.getElementById('loginEmail');
        const password      = document.getElementById('loginPassword');
        const emailFeedback = email.parentElement.querySelector('.invalid-feedback');
        const problemLink    = document.getElementById('passwordProblemLink');
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

        // Password help dialog
        (function initPasswordHelpDialog() {
            // Password help dialog element definition
            const dialog = document.getElementById('passwordHelpDialog');
            if (!dialog) return;

            const dialogEmail      = document.getElementById('passwordHelpEmail');
            const feedback         = document.getElementById('passwordHelpFeedback');
            const alertEl          = document.getElementById('passwordHelpAlert');
            const forgottenBtn     = document.getElementById('forgottenPasswordBtn');
            const expiredBtn       = document.getElementById('expiredPasswordBtn');

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

            // Send login credentials
            try {
                const res = await fetch('/api/login', {
                    method:  'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body:    JSON.stringify({
                        email:    email.value.trim(),
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
        bindLive(lastName,  () => validateName(lastName));
        bindLive(email,     validateEmail);
        bindLive(password,  validatePassword);
        bindLive(confirm,   validateConfirm);

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
        const dialog  = document.getElementById('logoutDialog');
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
        const elError   = document.getElementById('magic-error');
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

        const email          = document.getElementById('cpEmail');
        const current        = document.getElementById('cpCurrentPassword');
        const newPw          = document.getElementById('cpNewPassword');
        const confirm        = document.getElementById('cpConfirmPassword');
        const ruleList       = document.getElementById('cpRuleList');
        const strengthFill   = document.getElementById('cpStrengthFill');
        const strengthLabel  = document.getElementById('cpStrengthLabel');
        const emailFeedback  = document.getElementById('cpEmailFeedback');
        const alertEl        = document.getElementById('changePwAlert');
        const submitBtn      = document.getElementById('cpSubmitBtn');

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

        function validateNewPw()      { return mark(newPw, passwordMeetsAll(newPw.value)); }
        function validateConfirm()    { return mark(confirm, confirm.value.length > 0 && confirm.value === newPw.value); }

        // Password progress bar
        function updatePasswordUI() {
            renderStrength(ruleList, strengthFill, strengthLabel, newPw.value);
            if (confirm.value.length > 0) validateConfirm();
        }

        const bindLive = createLiveBinder();

        // Validation on every change
        bindLive(email,   validateEmail);
        bindLive(current, validateCurrent);
        bindLive(newPw,   validateNewPw);
        bindLive(confirm, validateConfirm);

        // Password bar update on change
        newPw.addEventListener('input', updatePasswordUI);
        newPw.addEventListener('blur',  updatePasswordUI);

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
            const results = [ validateEmail(), validateCurrent(), validateNewPw(), validateConfirm() ];
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
                    method:  'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body:    JSON.stringify({
                        email:           email.value.trim(),
                        currentPassword: current.value,
                        newPassword:     newPw.value
                    })
                });
                const data = await res.json().catch(() => ({}));

                email.value   = '';
                current.value = '';
                newPw.value   = '';
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

        const codeInput   = document.getElementById('tfCode');
        const targetEmail = document.getElementById('tfTargetEmail');
        const alertEl     = document.getElementById('tfAlert');
        const backLink    = document.getElementById('tfBackLink');

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
                    method:  'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body:    JSON.stringify({
                        pendingToken: pendingToken,
                        code:         codeInput.value.trim()
                    })
                });

                // Response check
                const data = await res.json().catch(() => ({}));

                if (!res.ok) {
                    const raw  = (data.message || '').toLowerCase();
                    const isSessionExpired  = res.status === 404 || data.code === 5 ||
                        raw.includes('session expired') ||
                        raw.includes('login session');
                    const isTooManyAttempts = raw.includes('too many');
                    const isCodeExpired     = raw.includes('code expired') ||
                        raw.includes('not found');
                    const isWrongCode       = raw.includes('do not match') ||
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
                    id:        claims.sub   || '',
                    email:     claims.email || pendingEmail || '',
                    firstName: '',
                    lastName:  '',
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
        const dialog  = document.getElementById('createFolderDialog');
        if (!trigger || !dialog) return;

        const input      = document.getElementById('folderNameInput');
        const confirmBtn = document.getElementById('createFolderConfirm');
        const alertEl    = document.getElementById('folderAlert');

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
                const qs = new URLSearchParams();
                qs.set('token', token);
                const res = await fetch('/api/storage/folder', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({
                        path:  newPath,
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
        const triggerPage   = document.getElementById('uploadButtonPage');
        const fileInput     = document.getElementById('uploadFileInput');
        const alertEl       = document.getElementById('uploadAlert');
        const fileInfo      = document.getElementById('uploadFileInfo');
        const fileNameEl    = document.getElementById('uploadFileName');
        const fileSizeEl    = document.getElementById('uploadFileSize');
        const progressWrap  = document.getElementById('uploadProgressWrap');
        const progressBar   = document.getElementById('uploadProgressBar');
        const progressText  = document.getElementById('uploadProgressText');
        const submitBtn     = document.getElementById('uploadFileConfirm');

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
        if (triggerPage)   triggerPage.addEventListener('click', openDialog);

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

            const token = localStorage.getItem('token');

            // Send file
            try {
                const res = await fetch('/api/storage/file', {
                    method:  'POST',
                    headers: token ? { 'Authorization': `Bearer ${token}` } : {},
                    body:    formData
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

    (function initFileList() {
        'use strict';

        const tbody = document.getElementById('fileTableBody');
        if (!tbody) return;

        const bc = document.getElementById('fileBreadcrumb');
        let currentPath = '';

        function formatSize(bytes) {
            if (!bytes || bytes <= 0) return '—';
            if (bytes < 1024) return bytes + ' B';
            if (bytes < 1024 ** 2) return (bytes / 1024).toFixed(1) + ' KB';
            if (bytes < 1024 ** 3) return (bytes / 1024 ** 2).toFixed(1) + ' MB';
            return (bytes / 1024 ** 3).toFixed(2) + ' GB';
        }

        function row(file) {
            const icon = file.isDirectory
                ? 'bi-folder-fill'
                : 'bi-file-earmark-fill';
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
                <button class="btn-icon" aria-label="More options">
                    <i class="bi bi-three-dots"></i>
                </button>
            </td>`;
            if (file.isDirectory) {
                tr.style.cursor = 'pointer';
                tr.addEventListener('click', (e) => {
                    if (e.target.closest('button')) return;
                    load(file.path);
                });
            }
            return tr;
        }

        function escapeHtml(s) {
            return s.replace(/[&<>"']/g, c => ({
                '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
            }[c]));
        }

        async function load(dirPath) {
            const token = localStorage.getItem('token');
            if (!token) { window.location.replace('login.html'); return; }

            tbody.innerHTML = `<tr><td colspan="5" class="text-soft p-3">Loading…</td></tr>`;

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

                // ── NEW: commit the path only after a successful load ──
                currentFolderPath = dirPath;
                renderBreadcrumb(dirPath);

                tbody.innerHTML = '';
                if (files.length === 0) {
                    tbody.innerHTML = `<tr><td colspan="5" class="text-soft p-3">This folder is empty.</td></tr>`;
                    return;
                }

                files
                    .slice()
                    .sort((a, b) => (b.isDirectory - a.isDirectory) || a.name.localeCompare(b.name))
                    .forEach(f => tbody.appendChild(row(f)));

            } catch (err) {
                tbody.innerHTML =
                    `<tr><td colspan="5" class="text-danger p-3">Failed to load files: ${escapeHtml(err.message)}</td></tr>`;
            }
        }

        load('');
        window.addEventListener('files-changed', () => load(currentFolderPath));
    })();
})();