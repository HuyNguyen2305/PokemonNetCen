package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	HOST         = "localhost"
	PORT         = "8081"
	TYPE         = "tcp"
	MIN_PLAYERS  = 2
	POKEDEX_FILE = "../assests/pokedex.json"
	USER_FILE    = "../assests/user.json"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Pokemon struct {
	Name     string   `json:"Name"`
	Elements []string `json:"Elements"`
	Stats    Stats    `json:"Stats"`
	Profile  Profile  `json:"Profile"`
	Damage   []Damage `json:"DamegeWhenAttacked"`
}

type Stats struct {
	HP         int `json:"HP"`
	Attack     int `json:"Attack"`
	Defense    int `json:"Defense"`
	Speed      int `json:"Speed"`
	Sp_Attack  int `json:"Sp_Attack"`
	Sp_Defense int `json:"Sp_Defense"`
}

type Profile struct {
	Height      float64 `json:"Height"`
	Weight      float64 `json:"Weight"`
	CatchRate   int     `json:"CatchRate"`
	GenderRatio struct {
		MaleRatio   float64 `json:"MaleRatio"`
		FemaleRatio float64 `json:"FemaleRatio"`
	} `json:"GenderRatio"`
	EggGroup   string `json:"EggGroup"`
	HatchSteps int    `json:"HatchSteps"`
	Abilities  string `json:"Abilities"`
}

type Damage struct {
	Element     string  `json:"Element"`
	Coefficient float64 `json:"Coefficient"`
}

type Player struct {
	Conn               net.Conn
	Name               string
	Pokemons           []Pokemon
	ActivePokemonIndex int
}

type Battle struct {
	Player1 *Player
	Player2 *Player
	Turn    int // 0 for Player1, 1 for Player2
}

var users []User

// Function to load users from a JSON file
func loadUsers(filename string) ([]User, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var userData struct {
		Users []User `json:"users"`
	}
	err = json.NewDecoder(file).Decode(&userData)
	if err != nil {
		return nil, err
	}

	return userData.Users, nil
}

// Function to load Pokémon from a JSON file
func loadPokemons(filename string) ([]Pokemon, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var pokemons []Pokemon
	err = json.NewDecoder(file).Decode(&pokemons)
	if err != nil {
		return nil, err
	}

	return pokemons, nil
}

func main() {
	var err error
	// Start TCP server
	listener, err := net.Listen("tcp", HOST+":"+PORT)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server is listening on", HOST+":"+PORT)

	var players []*Player

	// Load user data from JSON file
	users, err = loadUsers(USER_FILE)
	if err != nil {
		fmt.Println("Error loading user data:", err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		fmt.Println("Client connected from", conn.RemoteAddr().String())

		// Authenticate the user
		if !authenticate(conn) {
			fmt.Println("Authentication failed. Closing connection.")
			conn.Write([]byte("Authentication failed\n"))
			conn.Close()
			continue
		}

		// Add player to the list
		player := &Player{
			Conn: conn,
			Name: fmt.Sprintf("Player_%s", conn.RemoteAddr().String()),
		}
		players = append(players, player)

		// Check if we have two players
		if len(players) == 2 {
			fmt.Println("Two players connected. Starting the game...")

			// Load Pokémon data from file
			pokemons, err := loadPokemons(POKEDEX_FILE)
			if err != nil {
				log.Fatal("Error loading Pokémon data:", err)
			}

			// Randomly assign 3 Pokémon to each player
			rand.Shuffle(len(pokemons), func(i, j int) {
				pokemons[i], pokemons[j] = pokemons[j], pokemons[i]
			})

			// Divide the shuffled list into two sets for each player
			players[0].Pokemons = pokemons[:3]
			players[1].Pokemons = pokemons[3:6]

			// Choose starting Pokémon
			players[0].ActivePokemonIndex = chooseStartingPokemon(players[0])
			players[1].ActivePokemonIndex = chooseStartingPokemon(players[1])

			// Start the battle
			pokemonBattle(&Battle{Player1: players[0], Player2: players[1]})
			break // Exit loop after starting the battle
		} else {
			fmt.Println("Waiting for another player to connect...")
		}
	}
}

func authenticate(conn net.Conn) bool {
	// Read authentication data from the connection
	authData := readFromConn(conn)
	parts := strings.Split(authData, "_")

	// Check if the format is correct
	if len(parts) != 2 {
		log.Printf("Authentication data format error: expected 2 parts, got %d", len(parts))
		return false
	}

	username := parts[0]
	receivedPassword := parts[1]

	// Load users from the JSON file
	users, err := loadUsers(USER_FILE)
	if err != nil {
		log.Printf("Error loading users: %v", err)
		return false
	}

	// Check each user for a match
	for _, user := range users {
		if user.Username == username && user.Password == receivedPassword {
			log.Println("Authentication successful")
			conn.Write([]byte("authenticated\n"))
			return true
		}
	}

	log.Println("Authentication failed: no matching user found")
	return false
}

func readFromConn(conn net.Conn) string {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Println("Error reading:", err)
		return ""
	}
	return strings.TrimSpace(string(buffer[:n]))
}

func chooseStartingPokemon(player *Player) int {
	// Display all available Pokémon options
	for i, pokemon := range player.Pokemons {
		player.Conn.Write([]byte(fmt.Sprintf("%d: %s\n", i+1, pokemon.Name)))
	}

	player.Conn.Write([]byte("Choose your starting Pokémon:\n"))

	for {
		choice := readFromConn(player.Conn)
		index, err := strconv.Atoi(choice)
		if err == nil && index >= 1 && index <= len(player.Pokemons) {
			return index - 1
		}
		player.Conn.Write([]byte("Invalid choice. Please choose a valid Pokémon.\n"))
	}
}

func pokemonBattle(battle *Battle) {
	player1 := battle.Player1
	player2 := battle.Player2

	for {
		// Determine current player based on speed
		currentPlayer := player1
		opponent := player2
		if battle.Player1.Pokemons[battle.Player1.ActivePokemonIndex].Stats.Speed <
			battle.Player2.Pokemons[battle.Player2.ActivePokemonIndex].Stats.Speed {
			currentPlayer, opponent = opponent, currentPlayer
		}

		// Prompt current player for action
		currentPlayer.Conn.Write([]byte("Your turn! Choose an action: attack, switch, or surrender\n"))
		action := readFromConn(currentPlayer.Conn)

		switch action {
		case "attack":
			performAttack(currentPlayer, opponent)
		case "switch":
			switchPokemon(currentPlayer)
		case "surrender":
			endBattle(battle, opponent)
			return
		default:
			currentPlayer.Conn.Write([]byte("Invalid action. Please choose attack, switch, or surrender.\n"))
			continue
		}

		// Check if opponent's Pokémon is defeated
		if opponent.Pokemons[opponent.ActivePokemonIndex].Stats.HP <= 0 {
			// Switch to next available Pokémon
			nextAvailable := false
			for i, pokemon := range opponent.Pokemons {
				if pokemon.Stats.HP > 0 {
					opponent.ActivePokemonIndex = i
					nextAvailable = true
					break
				}
			}
			if !nextAvailable {
				// No Pokémon left, opponent loses
				endBattle(battle, currentPlayer)
				return
			}
		}

		// Switch turn to the other player
		if currentPlayer == battle.Player1 {
			battle.Player1, battle.Player2 = battle.Player2, battle.Player1
		} else {
			battle.Player1, battle.Player2 = battle.Player1, battle.Player2
		}
	}
}

func performAttack(attacker, defender *Player) {
	// Get the active Pokémon for both players
	attackPokemon := &attacker.Pokemons[attacker.ActivePokemonIndex]
	defendPokemon := &defender.Pokemons[defender.ActivePokemonIndex]

	// Randomly choose between normal attack and special attack
	isSpecial := rand.Intn(2) == 0

	var damage int
	if isSpecial {
		// Calculate damage for special attack using Sp_Attack
		damage = (attackPokemon.Stats.Sp_Attack - defendPokemon.Stats.Sp_Defense)
	} else {
		// Calculate damage for normal attack using Attack
		damage = (attackPokemon.Stats.Attack - defendPokemon.Stats.Defense)
	}

	// Apply elemental damage coefficient from the defender's damage when attacked
	for _, damageInfo := range defendPokemon.Damage {
		if damageInfo.Element == attackPokemon.Elements[0] { // Assuming the first element is used for the attack
			damage = int(float64(damage) * damageInfo.Coefficient)
			break
		}
	}

	// Ensure minimum damage is 1
	if damage < 1 {
		damage = 1
	}

	// Update the defender's HP
	defendPokemon.Stats.HP -= damage
	if defendPokemon.Stats.HP < 0 {
		defendPokemon.Stats.HP = 0
	}

	// Send messages to both players
	attacker.Conn.Write([]byte(fmt.Sprintf("You attacked %s's %s for %d damage.\n", defender.Name, defendPokemon.Name, damage)))
	defender.Conn.Write([]byte(fmt.Sprintf("Your %s was attacked for %d damage.\n", defendPokemon.Name, damage)))
}

func switchPokemon(player *Player) {
	// Notify the player to choose a Pokémon to switch to
	player.Conn.Write([]byte("Choose a Pokémon to switch to:\n"))

	// List available Pokémon with their HP
	for i, pokemon := range player.Pokemons {
		player.Conn.Write([]byte(fmt.Sprintf("%d: %s (HP: %d)\n", i+1, pokemon.Name, pokemon.Stats.HP)))
	}

	// Loop until a valid choice is made
	for {
		choice := readFromConn(player.Conn) // Read the player's choice
		index, err := strconv.Atoi(choice)  // Convert choice to integer

		// Check if the choice is valid
		if err == nil && index >= 1 && index <= len(player.Pokemons) && player.Pokemons[index-1].Stats.HP > 0 {
			player.ActivePokemonIndex = index - 1 // Update the active Pokémon index
			player.Conn.Write([]byte(fmt.Sprintf("Switched to %s.\n", player.Pokemons[player.ActivePokemonIndex].Name)))
			return // Exit the function after a successful switch
		}

		// Notify the player of an invalid choice
		player.Conn.Write([]byte("Invalid choice. Please choose a valid Pokémon.\n"))
	}
}

func endBattle(battle *Battle, winner *Player) {
	// Notify the winner
	winner.Conn.Write([]byte("Congratulations! You won the battle.\n"))

	// Determine the loser
	loser := battle.Player1
	if winner == battle.Player1 {
		loser = battle.Player2
	}

	// Notify the loser
	loser.Conn.Write([]byte("You lost the battle.\n"))

	// Close connections
	if err := battle.Player1.Conn.Close(); err != nil {
		log.Printf("Error closing connection for %s: %v\n", battle.Player1.Name, err)
	}
	if err := battle.Player2.Conn.Close(); err != nil {
		log.Printf("Error closing connection for %s: %v\n", battle.Player2.Name, err)
	}
}
