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
        const forgotLink    = document.getElementById('forgotPasswordLink');

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

        forgotLink.addEventListener('click', (e) => {
            e.preventDefault();
            alert("Password reset button clicked!");
        });

        form.addEventListener('submit', function (e) {
            e.preventDefault();
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
            alert("Login button clicked!");
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

        form.addEventListener('submit', function (e) {
            e.preventDefault();
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
            alert("Create account clicked!");
        });

        [firstName, lastName, email, password, confirm].forEach(input => {
            input.addEventListener('input', () => {
                if (input.value.length === 0 && !touched.has(input)) clearMark(input);
            });
        });

        updatePasswordUI();
    })();
})();