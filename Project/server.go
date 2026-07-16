package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

// ==========================
// STRUCTURI DE DATE
// ==========================
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Action   string `json:"action"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type StartRequest struct {
	Username    string `json:"username"`
	ChallengeID int    `json:"challenge_id"`
}

type StartResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Port    int    `json:"port"`
}

type VerifyRequest struct {
	Username    string `json:"username"`
	ChallengeID int    `json:"challenge_id"`
	Flag        string `json:"flag"`
}

type UserStatusResponse struct {
	Success bool  `json:"success"`
	Score   int   `json:"score"`
	Solved  []int `json:"solved"`
}

// ==========================
// CONFIGURAȚIA CHALLENGE-URILOR
// Adaugă aici orice challenge nou: imaginea Docker folosită și
// câte puncte valorează. Trebuie sincronizat cu array-ul CHALLENGES
// din ctf.js (id-urile trebuie să coincidă).
// ==========================
var challengeImages = map[int]string{
	1: "os-ctf-chal1", // Sanity Check
	2: "os-ctf-chal2", // Cutia Pandorei
	3: "os-ctf-chal3", // Imaginea Vorbăreață
	4: "os-ctf-chal4", // Cifrul lui Cezar (ROT13)
	5: "os-ctf-chal5", // Codul Ascuns (Base64)
	6: "os-ctf-chal6", // Procesul Fantomă (variabilă de mediu)
	7: "os-ctf-chal7", // Deghizarea (extensie mincinoasă)
	8: "os-ctf-chal8", // Ușa Încuiată (sudo/permisiuni)
	9: "os-ctf-chal9", // Straturi (encoding multi-strat)
}

var challengePoints = map[int]int{
	1: 10,
	2: 20,
	3: 30,
	4: 40,
	5: 40,
	6: 50,
	7: 40,
	8: 30,
	9: 60,
}

// Aplică ROT13 (Cifrul lui Cezar cu deplasare 13) pe un text.
// Literele sunt rotite, orice alt caracter (cifre, {, }, _, spații) rămâne neschimbat.
func rot13(input string) string {
	rot := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		default:
			return r
		}
	}
	return strings.Map(rot, input)
}

// Construiește comanda de injectare a flag-ului, specifică fiecărui challenge.
// Rulează în interiorul containerului via 'docker exec'.
func buildInjectCommand(challengeID int, flag string) string {
	switch challengeID {
	case 1:
		// Sanity Check: flag-ul e direct într-un fișier text
		return fmt.Sprintf("echo '%s' > /home/student/bun_venit.txt", flag)
	case 2:
		// Cutia Pandorei: flag-ul e într-un fișier ASCUNS (începe cu punct),
		// dar cu un nume neutru care nu conține cuvântul "flag"
		// (.sys_cache.dat), alături de alte 4 fișiere "momeală" normale.
		// 'ls' fără -a nu-l arată, dar apare la 'unzip -l' / 'ls -a'.
		// Sursele stau în /opt/pandora_src (nu în /home/student),
		// deci nu sunt vizibile înainte ca arhiva să fie generată.
		return fmt.Sprintf(
			"sed -i 's/FLAG_PLACEHOLDER/%s/' /opt/pandora_src/.sys_cache.dat && "+
				"cd /opt/pandora_src && zip -j /home/student/misiune.zip readme.txt notes.txt config.yml access.log todo.md .sys_cache.dat >/dev/null && "+
				"chown student:student /home/student/misiune.zip",
			flag)
	case 3:
		// Imaginea Vorbăreață: flag-ul e adăugat ca text la finalul
		// fișierului JPG (nu afectează vizualizarea imaginii, dar
		// apare la 'strings imagine.jpg').
		return fmt.Sprintf("echo '%s' >> /home/student/imagine.jpg", flag)
	case 4:
		// Cifrul lui Cezar: un mesaj (instrucțiuni + flag) e criptat
		// integral cu ROT13 și salvat în mesaj_secret.txt. Trimitem
		// conținutul deja codat Base64 prin shell, ca să evităm
		// problemele de escaping cu newline-uri/caractere speciale.
		plain := fmt.Sprintf(
			"Salut! Acest mesaj este criptat cu ROT13 (Cifrul lui Cezar).\nDecodeaza-l ca sa gasesti flag-ul.\n\n%s\n",
			flag)
		encoded := rot13(plain)
		b64 := base64.StdEncoding.EncodeToString([]byte(encoded))
		return fmt.Sprintf(
			"echo '%s' | base64 -d > /home/student/mesaj_secret.txt && chown student:student /home/student/mesaj_secret.txt",
			b64)
	case 5:
		// Codul Ascuns: flag-ul codat simplu Base64, într-un fișier .b64
		b64 := base64.StdEncoding.EncodeToString([]byte(flag))
		return fmt.Sprintf(
			"echo '%s' > /home/student/secret.b64 && chown student:student /home/student/secret.b64",
			b64)
	case 6:
		// Procesul Fantomă: pornim un proces în fundal, deținut de
		// utilizatorul student, cu flag-ul într-o variabilă de mediu.
		// setsid + redirectare completă = procesul rămâne "daemon",
		// independent de sesiunea SSH curentă sau de docker exec.
		return fmt.Sprintf(
			`su - student -c "env FLAG='%s' setsid sleep infinity < /dev/null > /dev/null 2>&1 &"`,
			flag)
	case 7:
		// Deghizarea: un fișier numit poza.png, dar care e de fapt
		// text simplu - studentul trebuie să folosească 'file poza.png'
		// ca să descopere tipul real, apoi 'cat' pentru flag.
		// Trimis prin base64 ca să evităm probleme de escaping cu newline-uri.
		content := fmt.Sprintf("Raport confidential. Nu distribui acest fisier.\n%s\n", flag)
		b64content := base64.StdEncoding.EncodeToString([]byte(content))
		return fmt.Sprintf(
			"echo '%s' | base64 -d > /home/student/poza.png && chown student:student /home/student/poza.png",
			b64content)
	case 8:
		// Ușa Încuiată: /root/secret.txt e citibil doar de root (chmod 600),
		// dar regula din /etc/sudoers.d (definită static în Dockerfile.chal8)
		// îi permite lui 'student' să ruleze DOAR 'sudo cat /root/secret.txt'
		// fără parolă. La runtime doar populăm conținutul real al fișierului.
		return fmt.Sprintf(
			"echo '%s' > /root/secret.txt && chmod 600 /root/secret.txt && chown root:root /root/secret.txt",
			flag)
	case 9:
		// Straturi: flag-ul e codat pe 3 straturi succesive:
		// hex -> ROT13 -> Base64. Studentul trebuie să dea peel invers:
		// base64 -d, apoi tr ROT13 (auto-invers), apoi hex decode (xxd -r -p).
		hexEncoded := hex.EncodeToString([]byte(flag))
		rotated := rot13(hexEncoded)
		b64layers := base64.StdEncoding.EncodeToString([]byte(rotated))
		return fmt.Sprintf(
			"echo '%s' > /home/student/layers.txt && chown student:student /home/student/layers.txt",
			b64layers)
	default:
		return ""
	}
}

// ==========================
// FUNCȚII AJUTĂTOARE
// ==========================

func verifyFlagHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(w, r) {
		return
	}

	var req VerifyRequest
	json.NewDecoder(r.Body).Decode(&req)

	// 1. Verificăm flag-ul în baza de date
	var dbFlag string
	var solved bool
	err := db.QueryRow(`
		SELECT dynamic_flag, solved 
		FROM active_challenges ac
		JOIN users u ON ac.user_id = u.id
		WHERE u.username = ? AND ac.challenge_id = ?`,
		req.Username, req.ChallengeID).Scan(&dbFlag, &solved)

	resp := AuthResponse{}

	if err != nil {
		resp.Success = false
		resp.Message = "Nu ai o instanță activă pentru acest challenge."
	} else if solved {
		resp.Success = false
		resp.Message = "Ai rezolvat deja acest challenge! Poți relua provocarea oricând, dar punctele nu se mai adaugă a doua oară."
	} else if req.Flag == dbFlag {
		// 2. Flag corect! Actualizăm scorul și statusul
		points, ok := challengePoints[req.ChallengeID]
		if !ok {
			points = 10 // fallback de siguranță dacă challenge-ul nu e în map
		}

		db.Exec("UPDATE active_challenges SET solved = 1 WHERE user_id = (SELECT id FROM users WHERE username = ?) AND challenge_id = ?", req.Username, req.ChallengeID)
		db.Exec("UPDATE users SET score = score + ? WHERE username = ?", points, req.Username)

		resp.Success = true
		resp.Message = fmt.Sprintf("Flag corect! Ai primit %d puncte.", points)
		log.Printf("[FLAG] Utilizatorul %s a rezolvat Challenge %d! (+%d puncte)", req.Username, req.ChallengeID, points)
	} else {
		resp.Success = false
		resp.Message = "Flag incorect, mai încearcă!"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Generează un flag aleatoriu de tip ATM_CTF{...}
func generateDynamicFlag() string {
	bytes := make([]byte, 6) // Generăm 6 bytes aleatori
	rand.Read(bytes)
	return fmt.Sprintf("ATM_CTF{%s}", hex.EncodeToString(bytes))
}

// Permite cererile CORS (Cross-Origin) de la frontend
func setupCORS(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	return r.Method == "OPTIONS"
}

// ==========================
// RUTELE API
// ==========================

func authHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(w, r) {
		return
	}

	var req AuthRequest
	json.NewDecoder(r.Body).Decode(&req)
	req.Username = strings.TrimSpace(req.Username)

	if len(req.Username) < 3 || len(req.Password) < 3 {
		json.NewEncoder(w).Encode(AuthResponse{Success: false, Message: "Minim 3 caractere necesare."})
		return
	}

	var hashFromDB string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", req.Username).Scan(&hashFromDB)

	resp := AuthResponse{}

	if req.Action == "register" {
		if err == sql.ErrNoRows {
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			_, errInsert := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", req.Username, string(hashedPassword))

			if errInsert != nil {
				resp.Success = false
				resp.Message = "Eroare la baza de date."
			} else {
				resp.Success = true
				resp.Message = "Cont creat cu succes!"
				log.Printf("[SUCCES] Cont creat: %s", req.Username)
			}
		} else {
			resp.Success = false
			resp.Message = "Nume de utilizator folosit."
		}
	} else if req.Action == "login" {
		if err == sql.ErrNoRows {
			resp.Success = false
			resp.Message = "Contul nu există."
		} else {
			if bcrypt.CompareHashAndPassword([]byte(hashFromDB), []byte(req.Password)) != nil {
				resp.Success = false
				resp.Message = "Parolă incorectă!"
			} else {
				resp.Success = true
				resp.Message = "Autentificare reușită!"
				log.Printf("[SUCCES] Login reușit: %s", req.Username)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Returnează scorul curent și lista de challenge-uri rezolvate ale unui utilizator.
// Apelat de ctf.js la încărcarea dashboard-ului și după fiecare flag corect,
// ca sursa de adevăr să fie mereu baza de date, nu starea locală din JS.
func userStatusHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(w, r) {
		return
	}

	username := r.URL.Query().Get("username")
	resp := UserStatusResponse{}

	if username == "" {
		resp.Success = false
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	var score int
	err := db.QueryRow("SELECT score FROM users WHERE username = ?", username).Scan(&score)
	if err != nil {
		resp.Success = false
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	rows, err := db.Query(`
		SELECT ac.challenge_id 
		FROM active_challenges ac
		JOIN users u ON ac.user_id = u.id
		WHERE u.username = ? AND ac.solved = 1`, username)

	solved := []int{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			if rows.Scan(&cid) == nil {
				solved = append(solved, cid)
			}
		}
	}

	resp.Success = true
	resp.Score = score
	resp.Solved = solved

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Returnează portul cel mai mic, începând de la 2200, care NU e folosit
// de niciun container Docker activ în acest moment. Spre deosebire de
// varianta veche (bazată pe MAX(ssh_port) din DB), asta permite refolosirea
// porturilor eliberate după ce monitor.sh oprește și șterge un container.
func findFreePort() int {
	used := map[int]bool{}

	out, err := exec.Command("docker", "ps", "--format", "{{.Ports}}").Output()
	if err == nil {
		re := regexp.MustCompile(`:(\d+)->22/tcp`)
		for _, match := range re.FindAllStringSubmatch(string(out), -1) {
			if p, convErr := strconv.Atoi(match[1]); convErr == nil {
				used[p] = true
			}
		}
	} else {
		log.Printf("[AVERTISMENT] Nu am putut lista containerele active pentru calculul portului: %v", err)
	}

	port := 2200
	for used[port] {
		port++
	}
	return port
}

func startChallengeHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(w, r) {
		return
	}

	var req StartRequest
	json.NewDecoder(r.Body).Decode(&req)

	resp := StartResponse{}

	// 1. Luăm ID-ul utilizatorului din baza de date
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", req.Username).Scan(&userID)
	if err != nil {
		resp.Success = false
		resp.Message = "Utilizator invalid."
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Verificăm dacă challenge-ul are o imagine Docker configurată
	image, ok := challengeImages[req.ChallengeID]
	if !ok {
		resp.Success = false
		resp.Message = "Acest challenge nu este încă configurat pe server!"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	log.Printf("[DOCKER] Pregătesc instanța pentru %s (Challenge %d)", req.Username, req.ChallengeID)

	// 2. Găsim primul port liber, verificând containerele Docker
	// active în acest moment (nu istoricul din DB, care rămâne
	// cu porturi vechi chiar și după ce containerele au fost șterse).
	port := findFreePort()

	// 3. Generăm flag-ul
	flag := generateDynamicFlag()

	// 4. Pornim containerul Docker
	// Rulăm comanda: docker run -d -p PORT:22 os-ctf-chal1
	// Dacă portul e deja ocupat, docker run CREEAZĂ totuși containerul înainte
	// să eșueze la pornire, lăsând un container "orfan" în starea Created.
	// De aceea îl ștergem imediat (docker rm) înainte să încercăm portul următor.
	var out []byte
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		cmd := exec.Command("docker", "run", "-d", "--rm", "-p", fmt.Sprintf("%d:22", port), image)

		// IMPORTANT: folosim CombinedOutput ca să vedem și stderr, nu doar stdout.
		out, err = cmd.CombinedOutput()

		if err == nil {
			break // succes
		}

		outStr := strings.TrimSpace(string(out))
		log.Printf("[EROARE DOCKER] Portul %d indisponibil: %s", port, outStr)

		if strings.Contains(outStr, "port is already allocated") || strings.Contains(outStr, "Bind for") {
			// docker run a apucat să creeze containerul înainte să eșueze la start.
			// Extragem ID-ul (prima linie a output-ului) și îl ștergem, ca să nu
			// rămână containere "Created" acumulate la infinit.
			lines := strings.Split(outStr, "\n")
			if len(lines) > 0 {
				possibleID := strings.TrimSpace(lines[0])
				if len(possibleID) >= 12 {
					exec.Command("docker", "rm", "-f", possibleID).Run()
					log.Printf("[CLEANUP] Am șters containerul orfan %s creat pe portul ocupat %d", possibleID[:12], port)
				}
			}
			port++ // încercăm portul următor
			continue
		}

		// Altă eroare (imagine lipsă, permisiuni etc.) - nu are rost să retry-uim
		break
	}

	if err != nil {
		outStr := strings.TrimSpace(string(out))
		log.Printf("[EROARE DOCKER] Nu am putut porni containerul după %d încercări: %v | Output: %s", maxRetries, err, outStr)
		resp.Success = false
		resp.Message = "Eroare la pornirea instanței Docker: " + outStr
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Extragem ID-ul containerului (fără spații și enter-uri de la final)
	containerID := strings.TrimSpace(string(out))

	if containerID == "" {
		log.Printf("[EROARE DOCKER] Container ID gol - 'docker run' nu a returnat nimic.")
		resp.Success = false
		resp.Message = "Eroare: Docker nu a returnat un ID de container valid."
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	shortID := containerID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	log.Printf("[DOCKER] Container pornit: %s pe portul %d", shortID, port)

	// 5. Injectăm flag-ul specific fiecărui challenge (vezi buildInjectCommand)
	injectCmd := buildInjectCommand(req.ChallengeID, flag)
	if injectCmd != "" {
		injectOut, injectErr := exec.Command("docker", "exec", containerID, "bash", "-c", injectCmd).CombinedOutput()
		if injectErr != nil {
			// Nu oprim tot procesul, dar logăm eroarea ca să știm dacă flag-ul chiar a fost scris
			log.Printf("[EROARE INJECT FLAG] %v | Output: %s", injectErr, strings.TrimSpace(string(injectOut)))
		}
	}

	// 6. Salvăm totul în baza de date.
	// Folosim UPSERT: dacă userul mai are deja un rând pentru acest challenge
	// (l-a mai lansat înainte), îl actualizăm cu noul container/port/flag,
	// dar NU atingem coloana 'solved' - dacă era deja rezolvat, rămâne rezolvat,
	// iar punctele nu se vor mai aduna a doua oară la un submit ulterior.
	_, err = db.Exec(`
		INSERT INTO active_challenges (user_id, challenge_id, container_id, ssh_port, dynamic_flag) 
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, challenge_id) DO UPDATE SET
			container_id = excluded.container_id,
			ssh_port = excluded.ssh_port,
			dynamic_flag = excluded.dynamic_flag`,
		userID, req.ChallengeID, containerID, port, flag)

	if err != nil {
		log.Printf("[EROARE DB] Nu am putut salva datele instanței: %v", err)
	}

	// 7. Trimitem succesul către frontend
	resp.Success = true
	resp.Message = "Instanță pornită cu succes!"
	resp.Port = port

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./ctf_platform.db")
	if err != nil {
		log.Fatal("Eroare la deschiderea bazei de date:", err)
	}
	defer db.Close()

	// Creăm tabelele dacă nu există deja (prima rulare / DB proaspătă)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			score INTEGER NOT NULL DEFAULT 0
		)`)
	if err != nil {
		log.Fatal("Eroare la crearea tabelului users:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS active_challenges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			challenge_id INTEGER NOT NULL,
			container_id TEXT,
			ssh_port INTEGER,
			dynamic_flag TEXT,
			solved INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`)
	if err != nil {
		log.Fatal("Eroare la crearea tabelului active_challenges:", err)
	}

	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_active_challenges_user_challenge ON active_challenges(user_id, challenge_id)`)
	if err != nil {
		log.Printf("[AVERTISMENT] Nu am putut crea indexul unic (poate există deja date duplicate): %v", err)
	}

	http.HandleFunc("/api/auth", authHandler)
	http.HandleFunc("/api/start_challenge", startChallengeHandler)
	http.HandleFunc("/api/verify_flag", verifyFlagHandler)
	http.HandleFunc("/api/user_status", userStatusHandler)

	fmt.Println("========================================")
	fmt.Println("[*] Serverul OS-CTF este ON")
	fmt.Println("[*] Ascult pe portul 5000...")
	fmt.Println("========================================")

	log.Fatal(http.ListenAndServe(":5000", nil))
}