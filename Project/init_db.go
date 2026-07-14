package main

import (
	"database/sql"
	"fmt"
	"log"

	// Importăm driver-ul SQLite (underscore-ul este obligatoriu pentru drivere)
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 1. Conectarea la (sau crearea) bazei de date
	db, err := sql.Open("sqlite3", "./ctf_platform.db")
	if err != nil {
		log.Fatal("Eroare la deschiderea bazei de date:", err)
	}
	// Ne asigurăm că baza de date se închide la finalul funcției
	defer db.Close()

	// 2. Definirea tabelei Utilizatori
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		score INTEGER DEFAULT 0
	);`

	// 3. Definirea tabelei pentru Instanțele Docker / Challenge-uri
	activeChallengesTable := `
	CREATE TABLE IF NOT EXISTS active_challenges (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		challenge_id INTEGER,
		container_id TEXT,          /* ID-ul containerului Docker */
		ssh_port INTEGER,           /* Portul unic pentru student */
		dynamic_flag TEXT,          /* Flag-ul generat dinamic */
		solved BOOLEAN DEFAULT 0,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	// 4. Executarea comenzilor SQL
	_, err = db.Exec(usersTable)
	if err != nil {
		log.Fatal("Eroare la crearea tabelei users: ", err)
	}

	_, err = db.Exec(activeChallengesTable)
	if err != nil {
		log.Fatal("Eroare la crearea tabelei active_challenges: ", err)
	}

	fmt.Println("[+] Baza de date 'ctf_platform.db' a fost creată cu succes!")
	fmt.Println("[+] Tabelele 'users' și 'active_challenges' sunt gata.")
}
