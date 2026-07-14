package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
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
		resp.Message = "Ai rezolvat deja acest challenge!"
	} else if req.Flag == dbFlag {
		// 2. Flag corect! Actualizăm scorul și statusul
		db.Exec("UPDATE active_challenges SET solved = 1 WHERE user_id = (SELECT id FROM users WHERE username = ?) AND challenge_id = ?", req.Username, req.ChallengeID)
		db.Exec("UPDATE users SET score = score + 10 WHERE username = ?", req.Username) // Adăugăm 10 puncte

		resp.Success = true
		resp.Message = "Flag corect! Ai primit 10 puncte."
		log.Printf("[FLAG] Utilizatorul %s a rezolvat Challenge %d!", req.Username, req.ChallengeID)
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

	// Momentan am configurat doar Challenge 1 (Sanity Check)
	if req.ChallengeID != 1 {
		resp.Success = false
		resp.Message = "Acest challenge nu este încă configurat pe server!"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	log.Printf("[DOCKER] Pregătesc instanța pentru %s (Challenge %d)", req.Username, req.ChallengeID)

	// 2. Găsim următorul port liber (începem de la 2200)
	var maxPort sql.NullInt64
	db.QueryRow("SELECT MAX(ssh_port) FROM active_challenges").Scan(&maxPort)

	port := 2200
	if maxPort.Valid && maxPort.Int64 >= 2200 {
		port = int(maxPort.Int64) + 1
	}

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
		cmd := exec.Command("docker", "run", "-d", "-p", fmt.Sprintf("%d:22", port), "os-ctf-chal1")

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

	// 5. Injectăm flag-ul în fișier (Magia CTF-ului)
	injectCmd := fmt.Sprintf("echo '%s' > /home/student/bun_venit.txt", flag)
	injectOut, injectErr := exec.Command("docker", "exec", containerID, "bash", "-c", injectCmd).CombinedOutput()
	if injectErr != nil {
		// Nu oprim tot procesul, dar logăm eroarea ca să știm dacă flag-ul chiar a fost scris
		log.Printf("[EROARE INJECT FLAG] %v | Output: %s", injectErr, strings.TrimSpace(string(injectOut)))
	}

	// 6. Salvăm totul în baza de date
	_, err = db.Exec(`
		INSERT INTO active_challenges (user_id, challenge_id, container_id, ssh_port, dynamic_flag) 
		VALUES (?, ?, ?, ?, ?)`,
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

	http.HandleFunc("/api/auth", authHandler)
	http.HandleFunc("/api/start_challenge", startChallengeHandler) // RUTA NOUĂ PENTRU DOCKER
	http.HandleFunc("/api/verify_flag", verifyFlagHandler)

	fmt.Println("========================================")
	fmt.Println("[*] Serverul OS-CTF Backend este ON!")
	fmt.Println("[*] Ascult pe portul 5000...")
	fmt.Println("========================================")

	log.Fatal(http.ListenAndServe(":5000", nil))
}