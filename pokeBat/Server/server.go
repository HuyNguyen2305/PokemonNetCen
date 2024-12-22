package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	HOST         = "localhost"
	PORT         = "8080"
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

func randomPokemons(pokemons []Pokemon, count int) []Pokemon {
	rand.Shuffle(len(pokemons), func(i, j int) {
		pokemons[i], pokemons[j] = pokemons[j], pokemons[i]
	})
	return pokemons[:count]
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
			fmt.Println("Two players connected. Waiting for the game to start...")
			// Continue waiting for the next connection
			continue
		}

		// Load Pokémon data from file
		pokemons, err := loadPokemons(POKEDEX_FILE)
		if err != nil {
			log.Fatal("Error loading Pokémon data:", err)
		}

		// Randomly assign 3 Pokémon to each player
		player1 := players[0]
		player2 := players[1]

		rand.Shuffle(len(pokemons), func(i, j int) {
			pokemons[i], pokemons[j] = pokemons[j], pokemons[i]
		})

		// Divide the shuffled list into two sets for each player
		player1.Pokemons = pokemons[:3]
		player2.Pokemons = pokemons[3:6]

		// Choose starting Pokémon
		player1.ActivePokemonIndex = chooseStartingPokemon(player1)
		player2.ActivePokemonIndex = chooseStartingPokemon(player2)

		// Start the battle
		/*battle := &Battle{
			Player1: player1,
			Player2: player2,
		}*/

		//runBattle(battle)
		//break // Exit loop after starting the battle
	}
}

// Function to authenticate the user
func authenticate(conn net.Conn) bool {
	defer conn.Close() // Ensure the connection is closed when done
	fmt.Println("Authenticating client:", conn.RemoteAddr().String())

	// Read authentication data from client
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from client:", err)
		return false
	}

	authData := strings.TrimSpace(string(buffer[:n]))
	credentials := strings.Split(authData, "_")
	if len(credentials) != 2 {
		conn.Write([]byte("Invalid authentication format. Use 'username_password'.\n"))
		return false
	}

	username, password := credentials[0], credentials[1]
	if isAuthenticated(username, password) {
		conn.Write([]byte("authenticated\n"))
		fmt.Println("User  authenticated:", username)
		return true
	} else {
		conn.Write([]byte("authentication failed\n"))
		fmt.Println("Authentication failed for user:", username)
		return false
	}
}

// Function to check if the user is authenticated
func isAuthenticated(username, password string) bool {
	for _, user := range users {
		if user.Username == username && user.Password == password {
			return true
		}
	}
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
