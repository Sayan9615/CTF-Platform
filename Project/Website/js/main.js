document.addEventListener('DOMContentLoaded', () => {

    // --- Afișarea numelui de utilizator ---
    const urlParams = new URLSearchParams(window.location.search);
    const username = urlParams.get('username');

    const displayElement = document.getElementById('display-username');
    if (displayElement) {
        displayElement.innerHTML = username ? `👤 ${username}` : `👤 Guest`;
    }

    // --- Login: Autentificare (Comunicarea cu serverul Go) ---
    const loginForm = document.getElementById('login-form');
    if (loginForm) {
        let submitAction = 'login';

        const submitButtons = loginForm.querySelectorAll('button[type="submit"]');
        submitButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                submitAction = btn.value;
            });
        });

        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            const user = loginForm.elements['username'].value;
            const pass = loginForm.elements['password'].value;

            try {
                const response = await fetch('http://localhost:5000/api/auth', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username: user, password: pass, action: submitAction })
                });

                const result = await response.json();

                if (result.success) {
                    if (submitAction === 'login') {
                        window.location.href = `dashboard.html?username=${encodeURIComponent(user)}`;
                    } else {
                        alert("Cont creat cu succes! Acum te poți conecta folosind butonul 'Log In'.");
                        loginForm.reset();
                    }
                } else {
                    alert(result.message);
                }
            } catch (error) {
                console.error("Eroare:", error);
                alert("Nu m-am putut conecta la server. Asigură-te că server.go rulează!");
            }
        });
    }

    
    if (document.getElementById('challenge-grid') && typeof CTF !== 'undefined') {
        CTF.init();
    }
});