
const CTF = (() => {

    const API_BASE = 'http://localhost:5000';

   
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
                <p class="difficulty easy">${ch.difficulty === 'easy' ? 'Ușor' : ch.difficulty}</p>
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

        if (btnStart) btnStart.classList.remove('hidden');
        if (sshInfo) sshInfo.classList.add('hidden');
        if (flagInput) flagInput.value = '';

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
                sshCommandText.innerText = `ssh student@127.0.0.1 -p ${result.port}`;
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