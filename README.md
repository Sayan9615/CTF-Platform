# CTF-Platform

Platformă web pentru desfășurarea unor challenge-uri de tip **CTF (Capture The Flag)**, create special pentru exersarea conceptelor studiate la disciplina **Sisteme de Operare**.

Utilizatorii se autentifică, aleg un challenge din dashboard, pornesc o instanță izolată (container Docker) sau descarcă resursa necesară, și găsesc un **flag unic** folosind comenzi și instrumente specifice sistemului Linux.

---

## Arhitectură

```
┌─────────────────┐        HTTP (port 5000)        ┌──────────────────┐
│   Browser        │ ─────────────────────────────▶ │   server.go       │
│  (dashboard.html) │ ◀───────────────────────────── │  (Go + SQLite)    │
└─────────────────┘        JSON API + fișiere        └──────────────────┘
                                                              │
                                                              │ docker run / docker exec
                                                              ▼
                                                     ┌──────────────────┐
                                                     │  Containere       │
                                                     │  Docker (SSH)     │
                                                     │  câte unul per    │
                                                     │  instanță/user    │
                                                     └──────────────────┘
```

- **Frontend**: HTML/CSS/JS static (`Website/`), fără framework — `dashboard.html` randează dinamic lista de challenge-uri din `ctf.js`.
- **Backend**: un singur binar Go (`server.go`) care servește atât API-ul (`/api/...`), **cât și fișierele front-end** (nu mai e nevoie de alt webserver, ex. Python).
- **Bază de date**: SQLite (`ctf_platform.db`), creată automat la prima pornire — ține conturile, scorurile și starea fiecărui challenge per utilizator.
- **Izolare**: fiecare challenge SSH rulează într-un container Docker separat, cu flag generat dinamic la pornire și un port SSH unic alocat automat.
- **Auto-cleanup**: containerele au un script (`monitor.sh`) care oprește și șterge automat instanța când user-ul se deconectează (fără intervenție manuală).

---

## Structura proiectului

```
Project/
├── server.go              # backend Go (API + servire frontend)
├── ctf_platform.db         # baza de date SQLite (generată automat)
├── Website/
│   ├── index.html           # pagina de login
│   ├── dashboard.html        # dashboard cu challenge-uri
│   ├── css/style.css
│   ├── js/
│   │   ├── ctf.js            # definirea challenge-urilor + logica CTF
│   │   └── main.js            # login/auth + inițializare dashboard
│   └── assets/
│       └── CTF_MAP.zip        # resursă pentru challenge-ul Minecraft
├── chal1_sanity/            # Dockerfile + monitor.sh pentru fiecare challenge
├── chal2_pandora/
├── chal3_imagine/
├── ...
└── chal17_quiz/
```

Fiecare folder `chalN_*` conține `Dockerfile` + `monitor.sh` folosit la build-ul imaginii `os-ctf-chalN`.

---

## Instalare & rulare

### Cerințe
- Go (1.20+)
- Docker
- SQLite3 (`go-sqlite3`, se instalează automat la `go run`/`go build`)

### 1. Construiește imaginile Docker (o singură dată, pentru fiecare challenge)

```bash
cd chal1_sanity/ && docker build -f Dockerfile -t os-ctf-chal1 . && cd ..
cd chal2_pandora/ && docker build -f Dockerfile -t os-ctf-chal2 . && cd ..
# ... la fel pentru chal3 → chal9
```

### 2. Pornește serverul

Din folderul care conține atât `server.go`, cât și `Website/`:

```bash
sudo go run server.go
```

> `sudo` e necesar dacă utilizatorul curent nu e în grupul `docker` (`docker run`/`docker exec` cer acces la `/var/run/docker.sock`). Alternativ: `sudo usermod -aG docker $USER` + relogare, apoi rulezi fără `sudo`.

Serverul pornește pe portul **5000** și servește tot (API + interfață).

### 3. Deschide platforma

```
http://localhost:5000/index.html
```

---

## Conectare din rețea (mai mulți utilizatori)

Platforma nu e limitată la mașina pe care rulează serverul — oricine din aceeași rețea locală se poate conecta.

**1. Află IP-ul mașinii care rulează `server.go`:**
```bash
hostname -I
```
(ex: `172.20.10.2`)

**2. Deschide portul în firewall** (o singură dată):
```bash
sudo ufw allow 5000/tcp
sudo ufw allow 2200:2300/tcp   # range-ul de porturi SSH alocate instanțelor
```

**3. Ceilalți utilizatori accesează** din browser, folosind structura de mai sus în loc de `localhost`:
```
IP_utilizator_care_deschide_server.go:5000
```

Frontend-ul (`ctf.js`) detectează automat adresa serverului din URL (`window.location.hostname`), deci **nu trebuie modificat nimic în cod** pentru asta — funcționează identic indiferent de IP-ul folosit.

> Notă: dacă serverul rulează într-o mașină virtuală (ex. VirtualBox), adaptorul de rețea trebuie setat pe **Bridged** (nu NAT), altfel VM-ul nu e vizibil din restul rețelei.

---

## Cum funcționează un challenge (flux tipic)

1. Studentul apasă **"Lansează Instanța"** → frontend-ul trimite `POST /api/start_challenge`.
2. Serverul găsește un port SSH liber, pornește un container Docker nou din imaginea challenge-ului, generează un **flag unic** (`ATM_CTF{...}`) și îl injectează în container (fișier, arhivă, proces etc., diferit pentru fiecare challenge).
3. Studentul primește comanda SSH (`ssh student@<ip> -p <port>`, parola `student`), se conectează și investighează sistemul.
4. Găsește flagul și îl trimite prin formularul din dashboard → `POST /api/verify_flag`.
5. Dacă e corect, scorul se actualizează și challenge-ul se marchează **"Rezolvat"**. Poate fi reluat oricând, dar punctele nu se mai adaugă a doua oară.
6. Când studentul se deconectează de la SSH, containerul se oprește și se șterge automat.

---

## Challenge-uri implementate

| # | Nume | Concept SO | Dificultate | Puncte |
|---|------|-----------|:---:|:---:|
| 1 | Sanity Check | comenzi de bază, navigare în sistemul de fișiere | Ușor | 10 |
| 2 | Cutia Pandorei | arhive, fișiere ascunse | Ușor | 20 |
| 3 | Imaginea Vorbăreață | analiză de fișiere binare (`strings`) | Ușor | 30 |
| 4 | Cifrul Cezarului (ROT13) | procesare text în shell | Ușor | 40 |
| 5 | Șirul Bazei 64 | codare/decodare | Ușor | 40 |
| 6 | Procesul Fantomă | procese, `/proc`, variabile de mediu | Mediu | 50 |
| 7 | Deghizarea | steganografie, identificarea tipului real al unui fișier | Ușor | 40 |
| 8 | Ușa Încuiată | permisiuni, `sudo`, ownership | Ușor | 30 |
| 9 | Straturi | codare pe mai multe niveluri (hex/ROT13/Base64) | Mediu | 60 |
| 10 | M1n3cr4ft | challenge bonus (hartă statică, fără server dedicat) | Greu | 100 |
| 11 | Arhiva Blocată |arhive protejate cu parolă, brute-force (fcrackzip) | Mediu | 60 |
| 12 | Fișierul din Fișier| forensics, fișiere ascunse în imagini | Ușor | 40 |
| 13 | Baza Uitată | interogare baze de date SQLite `(sqlite3)` | Ușor | 40 |
| 14 | Crackme | reverse engineering de bază, analiză executabile | Ușor | 40 |
| 15 | Sparge Hash-ul | criptografie, spargere hash-uri MD5 (hashcat / john) | Mediu | 45 |
| 16 | Capturat în Trafic | forensics, analiză de trafic de rețea (tshark / tcpdump) | Ușor | 40 |
| 17 | Chestionarul Spatial | reverse engineering, analiză dinamică (ltrace), logică de input | Greu | 100

---

## Cum adaugi un challenge nou

1. **Definește-l** în `Website/js/ctf.js`, în array-ul `CHALLENGES` (id, titlu, categorie, puncte, dificultate, descriere).
2. **Creează imaginea Docker** (`Dockerfile` + `monitor.sh`, după modelul celorlalte).
3. **Înregistrează-l** în `server.go`:
   - adaugă imaginea în `challengeImages`
   - adaugă punctajul în `challengePoints`
   - adaugă un `case` nou în `buildInjectCommand` cu logica de injectare a flagului, specifică challenge-ului
4. Build imagine + restart server.

---

## Sistem de scor

- Fiecare utilizator are un scor total, actualizat la fiecare flag corect.
- Un challenge rezolvat rămâne marcat "Rezolvat" definitiv — poate fi refăcut oricând (util pentru practică), dar punctele se acordă o singură dată.
- Scorul și lista challenge-urilor rezolvate se sincronizează mereu cu baza de date (`GET /api/user_status`), nu se bazează pe stare locală din browser.