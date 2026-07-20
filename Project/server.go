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
	"time"

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
	1:  10,
	2:  20,
	3:  30,
	4:  40,
	5:  40,
	6:  50,
	7:  40,
	8:  30,
	9:  60,
	10: 100,
}


var staticFlags = map[int]string{
	10: "ATM_CTF{m1n3cr47t_is_f4n_3}",
}


var minecraftChallenges = map[int]bool{}


const (
	mcSignX = -965
	mcSignY = 14
	mcSignZ = -611
)


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


func buildInjectCommand(challengeID int, flag string) string {
	switch challengeID {
	case 1:
		
		return fmt.Sprintf("echo '%s' > /home/student/bun_venit.txt", flag)
	case 2:
		
		return fmt.Sprintf(
			"sed -i 's/FLAG_PLACEHOLDER/%s/' /opt/pandora_src/.sys_cache.dat && "+
				"cd /opt/pandora_src && zip -j /home/student/misiune.zip readme.txt notes.txt config.yml access.log todo.md .sys_cache.dat >/dev/null && "+
				"chown student:student /home/student/misiune.zip",
			flag)
	case 3:
		
		return fmt.Sprintf("echo '%s' >> /home/student/imagine.jpg", flag)
	case 4:
		
		plain := fmt.Sprintf(
			"Salut! Acest mesaj este criptat cu ROT13 (Cifrul lui Cezar).\nDecodeaza-l ca sa gasesti flag-ul.\n\n%s\n",
			flag)
		encoded := rot13(plain)
		b64 := base64.StdEncoding.EncodeToString([]byte(encoded))
		return fmt.Sprintf(
			"echo '%s' | base64 -d > /home/student/mesaj_secret.txt && chown student:student /home/student/mesaj_secret.txt",
			b64)
	case 5:
		
		b64 := base64.StdEncoding.EncodeToString([]byte(flag))
		return fmt.Sprintf(
			"echo '%s' > /home/student/secret.b64 && chown student:student /home/student/secret.b64",
			b64)
	case 6:
		
		return fmt.Sprintf(
			`su - student -c "env FLAG='%s' setsid sleep infinity < /dev/null > /dev/null 2>&1 &"`,
			flag)
	case 7:
		
		content := fmt.Sprintf("Raport confidential. Nu distribui acest fisier.\n%s\n", flag)
		b64content := base64.StdEncoding.EncodeToString([]byte(content))
		return fmt.Sprintf(
			"echo '%s' | base64 -d > /home/student/poza.png && chown student:student /home/student/poza.png",
			b64content)
	case 8:
		
		return fmt.Sprintf(
			"echo '%s' > /root/secret.txt && chmod 600 /root/secret.txt && chown root:root /root/secret.txt",
			flag)
	case 9:
		
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


func injectMinecraftFlag(containerID string, flag string) error {
	const maxAttempts = 40 

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		out, err := exec.Command("docker", "exec", containerID, "rcon-cli", "list").CombinedOutput()
		if err == nil {
			lastErr = nil
			log.Printf("[MINECRAFT] RCON gata după %d încercări: %s", attempt+1, strings.TrimSpace(string(out)))
			break
		}
		lastErr = err
		time.Sleep(3 * time.Second)
	}
	if lastErr != nil {
		return fmt.Errorf("RCON nu a devenit disponibil în timp util: %w", lastErr)
	}

	
	forceloadOut, forceloadErr := exec.Command("docker", "exec", containerID, "rcon-cli",
		fmt.Sprintf("forceload add %d %d", mcSignX, mcSignZ)).CombinedOutput()
	if forceloadErr != nil {
		return fmt.Errorf("eroare la forceload pe chunk-ul semnului: %v | output: %s", forceloadErr, strings.TrimSpace(string(forceloadOut)))
	}
	log.Printf("[MINECRAFT] Forceload aplicat pe chunk-ul semnului: %s", strings.TrimSpace(string(forceloadOut)))

	
	nbtCommand := fmt.Sprintf(
		`data merge block %d %d %d {front_text:{messages:['"%s"','""','""','""']}}`,
		mcSignX, mcSignY, mcSignZ, flag)

	out, err := exec.Command("docker", "exec", containerID, "rcon-cli", nbtCommand).CombinedOutput()
	if err != nil {
		return fmt.Errorf("eroare la scrierea flagului pe semn: %v | output: %s", err, strings.TrimSpace(string(out)))
	}
	log.Printf("[MINECRAFT] Flag scris pe semn la (%d, %d, %d): %s", mcSignX, mcSignY, mcSignZ, strings.TrimSpace(string(out)))
	return nil
}



func verifyFlagHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(w, r) {
		return
	}

	var req VerifyRequest
	json.NewDecoder(r.Body).Decode(&req)

	resp := AuthResponse{}

	
	if staticFlag, isStatic := staticFlags[req.ChallengeID]; isStatic {
		var userID int
		err := db.QueryRow("SELECT id FROM users WHERE username = ?", req.Username).Scan(&userID)
		if err != nil {
			resp.Success = false
			resp.Message = "Utilizator invalid."
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		
		db.Exec(`
			INSERT INTO active_challenges (user_id, challenge_id, dynamic_flag)
			VALUES (?, ?, ?)
			ON CONFLICT(user_id, challenge_id) DO UPDATE SET dynamic_flag = excluded.dynamic_flag`,
			userID, req.ChallengeID, staticFlag)
	}

	
	var dbFlag string
	var solved bool
	err := db.QueryRow(`
		SELECT dynamic_flag, solved 
		FROM active_challenges ac
		JOIN users u ON ac.user_id = u.id
		WHERE u.username = ? AND ac.challenge_id = ?`,
		req.Username, req.ChallengeID).Scan(&dbFlag, &solved)

	if err != nil {
		resp.Success = false
		resp.Message = "Nu ai o instanță activă pentru acest challenge."
	} else if solved {
		resp.Success = false
		resp.Message = "Ai rezolvat deja acest challenge! Poți relua provocarea oricând, dar punctele nu se mai adaugă a doua oară."
	} else if req.Flag == dbFlag {
		
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


func findFreePort(startFrom int) int {
	used := map[int]bool{}

	out, err := exec.Command("docker", "ps", "--format", "{{.Ports}}").Output()
	if err == nil {
		re := regexp.MustCompile(`:(\d+)->\d+/(tcp|udp)`)
		for _, match := range re.FindAllStringSubmatch(string(out), -1) {
			if p, convErr := strconv.Atoi(match[1]); convErr == nil {
				used[p] = true
			}
		}
	} else {
		log.Printf("[AVERTISMENT] Nu am putut lista containerele active pentru calculul portului: %v", err)
	}

	port := startFrom
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

	
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", req.Username).Scan(&userID)
	if err != nil {
		resp.Success = false
		resp.Message = "Utilizator invalid."
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	
	image, ok := challengeImages[req.ChallengeID]
	if !ok {
		resp.Success = false
		resp.Message = "Acest challenge nu este încă configurat pe server!"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	log.Printf("[DOCKER] Pregătesc instanța pentru %s (Challenge %d)", req.Username, req.ChallengeID)

	
	internalPort := 22
	portRangeStart := 2200
	if minecraftChallenges[req.ChallengeID] {
		internalPort = 25565
		portRangeStart = 30000
	}
	port := findFreePort(portRangeStart)

	
	flag := generateDynamicFlag()

	
	var out []byte
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		cmd := exec.Command("docker", "run", "-d", "--rm", "-p", fmt.Sprintf("%d:%d", port, internalPort), image)

		
		out, err = cmd.CombinedOutput()

		if err == nil {
			break // succes
		}

		outStr := strings.TrimSpace(string(out))
		log.Printf("[EROARE DOCKER] Portul %d indisponibil: %s", port, outStr)

		if strings.Contains(outStr, "port is already allocated") || strings.Contains(outStr, "Bind for") {
			
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

	
	if minecraftChallenges[req.ChallengeID] {
		if injErr := injectMinecraftFlag(containerID, flag); injErr != nil {
			log.Printf("[EROARE INJECT FLAG MINECRAFT] %v", injErr)
		}

		
		go func(cid string) {
			time.Sleep(30 * time.Minute)
			exec.Command("docker", "stop", cid).Run()
			shortCid := cid
			if len(shortCid) > 12 {
				shortCid = shortCid[:12]
			}
			log.Printf("[MINECRAFT] Instanța %s oprită automat după 30 de minute", shortCid)
		}(containerID)
	} else {
		injectCmd := buildInjectCommand(req.ChallengeID, flag)
		if injectCmd != "" {
			injectOut, injectErr := exec.Command("docker", "exec", containerID, "bash", "-c", injectCmd).CombinedOutput()
			if injectErr != nil {
				
				log.Printf("[EROARE INJECT FLAG] %v | Output: %s", injectErr, strings.TrimSpace(string(injectOut)))
			}
		}
	}

	
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

	
	http.Handle("/", http.FileServer(http.Dir("./Website")))

	fmt.Println("========================================")
	fmt.Println("[*] Serverul OS-CTF")
	fmt.Println("========================================")

	log.Fatal(http.ListenAndServe(":5000", nil))
}