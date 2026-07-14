
let currentChallengeId = null;


document.addEventListener('DOMContentLoaded', () => {

    // --- Dashboard: Preluarea și afișarea numelui de utilizator ---
    const urlParams = new URLSearchParams(window.location.search);
    const username = urlParams.get('username');

    const displayElement = document.getElementById('display-username');
    if (displayElement) {
        if (username) {
            displayElement.innerHTML = `👤 ${username}`;
        } else {
            displayElement.innerHTML = `👤 Guest`;
        }
    }

    // --- Login: Autentificare (Comunicarea cu serverul Go) ---
    const loginForm = document.getElementById('login-form');
    if (loginForm) {
        let submitAction = 'login';

        // Ascultăm click-urile pentru a ști dacă vrea "Login" sau "Register"
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
                        // Dacă e login reușit, mergem la dashboard
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
});


// ==========================================
// 2. Funcții globale pentru Modale și Challenge-uri (Dashboard)
// ==========================================

// Deschide fereastra pentru un anumit challenge și îi setează textele
function openModal(challengeId, title, description) {
    currentChallengeId = challengeId;

    // Setăm titlul și descrierea dinamic
    document.getElementById('modal-title').innerText = title;
    document.getElementById('modal-desc').innerHTML = description;

    // Resetăm interfața modalului (Ascundem SSH-ul, afișăm butonul de Start)
    const btnStart = document.getElementById('btn-start-instance');
    const sshInfo = document.getElementById('ssh-info');
    const flagInput = document.getElementById('flag-input');

    if (btnStart) btnStart.classList.remove('hidden');
    if (sshInfo) sshInfo.classList.add('hidden');
    if (flagInput) flagInput.value = '';

    // Afișăm modalul pe ecran
    document.getElementById('challenge-modal').classList.remove('hidden');
}

// Închide fereastra modală
function closeModal() {
    document.getElementById('challenge-modal').classList.add('hidden');
    currentChallengeId = null;
}

// --- Start Instanță Docker (Comunicare cu Backend) ---
async function startInstance() {
    const urlParams = new URLSearchParams(window.location.search);
    const username = urlParams.get('username');

    if (!username || username === 'Guest') {
        alert("Trebuie să fii logat pentru a lansa o instanță!");
        return;
    }

    if (!currentChallengeId) {
        alert("Nu s-a putut identifica challenge-ul selectat. Închide și redeschide fereastra.");
        return;
    }

    // Schimbăm UI-ul instant
    document.getElementById('btn-start-instance').classList.add('hidden');
    const sshInfo = document.getElementById('ssh-info');
    sshInfo.classList.remove('hidden');

    const sshCommandText = document.getElementById('ssh-command');
    sshCommandText.innerText = "Se creează mediul izolat... te rugăm așteaptă.";
    sshCommandText.style.color = "#ffcc00"; // Galben - loading

    try {
        const response = await fetch('http://localhost:5000/api/start_challenge', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                username: username,
                challenge_id: currentChallengeId
            })
        });

        // Verificăm codul HTTP înainte să parsăm JSON, ca să prindem erori 500 etc.
        if (!response.ok) {
            throw new Error(`Server a răspuns cu status ${response.status}`);
        }

        const result = await response.json();

        if (result.success) {
            sshCommandText.innerText = `ssh student@127.0.0.1 -p ${result.port}`;
            sshCommandText.style.color = "#00ffcc"; // Verde/Cyan - succes
        } else {
            // Afișăm eroarea inclusiv în caseta SSH, nu doar în alert,
            // ca să vezi mesajul detaliat trimis acum de server.go
            sshCommandText.innerText = "Eroare: " + result.message;
            sshCommandText.style.color = "#ff5555";
            alert("Eroare de la server: " + result.message);
            document.getElementById('btn-start-instance').classList.remove('hidden');
        }
    } catch (error) {
        console.error("Eroare la pornirea instanței:", error);
        sshCommandText.innerText = "Eroare critică: " + error.message;
        sshCommandText.style.color = "#ff5555";
        alert("Eroare critică: Nu am putut contacta serverul!");
        document.getElementById('btn-start-instance').classList.remove('hidden');
    }
}

// --- Validare Flag (Comunicare cu Backend) ---
async function submitFlag(event) {
    event.preventDefault();

    const flagInput = document.getElementById('flag-input');
    const userFlag = flagInput ? flagInput.value : '';

    if (!userFlag) return;

    const urlParams = new URLSearchParams(window.location.search);
    const username = urlParams.get('username');

    if (!username || username === 'Guest') {
        alert("Trebuie să fii logat pentru a trimite un flag!");
        return;
    }

    try {
        const response = await fetch('http://localhost:5000/api/verify_flag', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                username: username,
                challenge_id: currentChallengeId,
                flag: userFlag
            })
        });

        const result = await response.json();

        if (result.success) {
            alert("🎉 Corect! Flag-ul a fost validat. (Urmează să actualizăm și punctajul)");
            closeModal();
        } else {
            alert("❌ Flag incorect: " + result.message);
        }
    } catch (error) {
        console.error("Eroare la verificarea flag-ului:", error);
        alert("Eroare de conexiune la server.");
    }
}