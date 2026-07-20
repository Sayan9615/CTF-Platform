const CTF = (() => {

    const API_BASE = '';

    
    const CHALLENGES = [
        {
            id: 1,
            title: 'Sanity Check',
            category: 'Comenzi de bază',
            points: 10,
            difficulty: 'easy',
            description: 'Pentru a te obișnui cu platforma, conectează-te la instanță și listează fișierele din folderul tău.'
        },
        {
            id: 2,
            title: 'Cutia Pandorei',
            category: 'Arhive & Ascunse',
            points: 20,
            difficulty: 'easy',
            description: 'Pe server se află o arhivă numită <b>misiune.zip</b>. Extrage arhiva (<code>unzip misiune.zip</code>) și cercetează cu atenție conținutul fișierelor - flag-ul e ascuns printre ele.'
        },
        {
            id: 3,
            title: 'Imaginea Vorbăreață',
            category: 'Analiză Text',
            points: 30,
            difficulty: 'easy',
            description: 'Ai primit pe server un fișier suspect numit <b>imagine.jpg</b>. Uneori imaginile "vorbesc" mai mult decât par - încearcă să extragi textul din interiorul fișierului.'
        },
        {
           id: 4,
           title: 'Cifrul Cezarului (ROT13)' ,
           category: 'Criptografie',
           points: 40,
           difficulty: 'easy',
           description: 'Pe server se află un fișier numit <b>mesaj_secret.txt</b>. Textul din interior a fost codat folosind cifrul ROT13. Citește fișierul și decodează mesajul (hint: poți folosi utilitarul <code>tr</code> din Linux).'
        },
        {
           id: 5,
           title: 'Șirul Bazei 64' ,
           category: 'Criptografie',
           points: 40,
           difficulty: 'easy',
           description: 'În directorul tău home vei găsi un fișier numit <b>secret.b64</b>. Conținutul său nu poate fi citit direct pentru că este codat în format Base64. Folosește comanda potrivită în terminal pentru a decoda șirul și a obține flag-ul.'
        },
        {
           id: 6,
           title: 'Procesul Fantomă' ,
           category: 'Administrare Procese',
           points: 50,
           difficulty: 'medium',
           description: 'Niciun fișier de data aceasta! Pe server rulează un proces suspect în fundal (background). Folosește utilitare de monitorizare a sistemului (cum ar fi <code>ps aux</code> sau <code>top</code>) pentru a inspecta procesele active. Flag-ul este ascuns chiar în comanda care a lansat acel proces fantomă.'
        },
        {
           id: 7,
           title: 'Deghizarea',
           category: 'Steganografie',
           points: 40,
           difficulty: 'easy',
           description: 'Ai primit fișierul <b>poza.png</b>. Ceva nu e în regulă cu el - verifică ce tip de fișier este de fapt (nu te lua doar după extensie).'
        },
        {
           id: 8,
           title: 'Ușa Încuiată',
           category: 'Permisiuni & Utilizatori',
           points: 30,
           difficulty: 'easy',
           description: 'Există un fișier <b>/root/secret.txt</b> pe care doar root îl poate citi direct. Verifică ce comenzi cu privilegii de root ai voie să rulezi (<code>sudo -l</code>).'
        },
        {
           id: 9,
           title: 'Straturi',
           category: 'Criptografie',
           points: 60,
           difficulty: 'medium',
           description: 'Fișierul <b>layers.txt</b> conține flag-ul, dar codat pe mai multe straturi succesive. Decodează pas cu pas, ca la o ceapă - verifică după fiecare pas ce fel de date obții.'
        },
        {
           id: 10,
           title: 'M1n3cr4ft',
           category: 'Misc',
           points: 100,
           difficulty: 'hard',
           type: 'download',
           downloadUrl: 'assets/CTF_MAP.zip',
           description: 'Descarcă harta de mai jos și deschide-o local, în Minecraft (versiunea 26.2 singleplayer). Demonstrează-ți capacitățile de aventurier și caută flag-ul.'
        }
    ];

    let currentChallengeId = null;
    let solvedIds = new Set();

    function renderGrid() {
        const grid = document.getElementById('challenge-grid');
        if (!grid) return;

        grid.innerHTML = '';

        CHALLENGES.forEach(ch => {
            const isSolved = solvedIds.has(ch.id);

            const card = document.createElement('div');
            card.className = 'challenge-card available' + (isSolved ? ' solved' : '');
            card.addEventListener('click', () => openModal(ch.id));

            card.innerHTML = `
                <div class="card-header">
                    <span class="category">${ch.category}</span>
                    <span class="points">${ch.points} pts</span>
                </div>
                <h3>${ch.title}</h3>
                <p class="difficulty ${ch.difficulty}" ${ch.difficulty === 'hard' ? 'style="color:#ff3b3b;font-weight:bold;"' : ''}>${ch.difficulty === 'easy' ? 'Ușor' : ch.difficulty === 'medium' ? 'Mediu' : 'Greu'}</p>
                <div class="status">${isSolved ? '✅ Rezolvat' : '▶️ Disponibil'}</div>
            `;

            grid.appendChild(card);
        });
    }

    function openModal(challengeId) {
        const ch = CHALLENGES.find(c => c.id === challengeId);
        if (!ch) return;

        currentChallengeId = challengeId;

        document.getElementById('modal-title').innerText = ch.title;
        document.getElementById('modal-desc').innerHTML = ch.description;

        const btnStart = document.getElementById('btn-start-instance');
        const sshInfo = document.getElementById('ssh-info');
        const flagInput = document.getElementById('flag-input');

        
        if (sshInfo) {
            sshInfo.classList.add('hidden');
            sshInfo.style.display = 'none';
        }
        if (flagInput) flagInput.value = '';

        if (ch.type === 'download') {
           
            if (btnStart) {
                btnStart.classList.remove('hidden');
                btnStart.innerText = '⬇️ Descarcă Harta';
                btnStart.onclick = () => {
                    window.location.href = ch.downloadUrl;
                };
            }
        } else {
            
            if (btnStart) {
                btnStart.classList.remove('hidden');
                btnStart.innerText = 'Lansează Instanța';
                btnStart.onclick = () => startInstance();
            }
        }

        document.getElementById('challenge-modal').classList.remove('hidden');
    }

    function closeModal() {
        document.getElementById('challenge-modal').classList.add('hidden');
        currentChallengeId = null;
    }

    async function startInstance() {
        const username = getUsername();

        if (!username || username === 'Guest') {
            alert("Trebuie să fii logat pentru a lansa o instanță!");
            return;
        }
        if (!currentChallengeId) {
            alert("Nu s-a putut identifica challenge-ul selectat. Închide și redeschide fereastra.");
            return;
        }

        document.getElementById('btn-start-instance').classList.add('hidden');
        const sshInfo = document.getElementById('ssh-info');
        sshInfo.classList.remove('hidden');
        sshInfo.style.display = ''; // anulăm forțarea display:none din openModal

        const instructionsP = sshInfo.querySelector('p');
        if (instructionsP) {
            instructionsP.innerHTML = 'Folosește comanda de mai jos în terminalul tău pentru a te conecta (Parola este <strong>student</strong>):';
        }

        const sshCommandText = document.getElementById('ssh-command');
        sshCommandText.innerText = "Se creează mediul izolat... te rugăm așteaptă.";
        sshCommandText.style.color = "#ffcc00";

        try {
            const response = await fetch(`${API_BASE}/api/start_challenge`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, challenge_id: currentChallengeId })
            });

            if (!response.ok) throw new Error(`Server a răspuns cu status ${response.status}`);

            const result = await response.json();

            if (result.success) {
                sshCommandText.innerText = `ssh student@${window.location.hostname} -p ${result.port}`;
                sshCommandText.style.color = "#00ffcc";
            } else {
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

    async function submitFlag(event) {
        event.preventDefault();

        const flagInput = document.getElementById('flag-input');
        const userFlag = flagInput ? flagInput.value : '';
        if (!userFlag) return;

        const username = getUsername();
        if (!username || username === 'Guest') {
            alert("Trebuie să fii logat pentru a trimite un flag!");
            return;
        }

        try {
            const response = await fetch(`${API_BASE}/api/verify_flag`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    username,
                    challenge_id: currentChallengeId,
                    flag: userFlag
                })
            });

            const result = await response.json();

            if (result.success) {
                alert("🎉 " + result.message);
                solvedIds.add(currentChallengeId);
                closeModal();
                renderGrid();
                refreshScore();
            } else {
                alert("❌ " + result.message);
            }
        } catch (error) {
            console.error("Eroare la verificarea flag-ului:", error);
            alert("Eroare de conexiune la server.");
        }
    }

    async function refreshScore() {
        const username = getUsername();
        if (!username || username === 'Guest') return;

        try {
            const response = await fetch(`${API_BASE}/api/user_status?username=${encodeURIComponent(username)}`);
            const result = await response.json();

            if (result.success) {
                solvedIds = new Set(result.solved || []);

                const scoreEl = document.getElementById('user-score-value');
                if (scoreEl) scoreEl.innerText = result.score;

                renderGrid();
            }
        } catch (error) {
            console.error("Nu am putut prelua scorul:", error);
        }
    }

    function getUsername() {
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get('username');
    }

    function init() {
        renderGrid();
        refreshScore();
    }

    return {
        init,
        openModal,
        closeModal,
        startInstance,
        submitFlag
    };
})();

window.openModal = (id) => CTF.openModal(id);
window.closeModal = () => CTF.closeModal();
window.startInstance = () => CTF.startInstance();
window.submitFlag = (e) => CTF.submitFlag(e);


document.addEventListener('DOMContentLoaded', () => {
    CTF.init();
});