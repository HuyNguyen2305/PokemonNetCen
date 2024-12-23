package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ---- Pokemon stats and structs stored here as well as other structs
// User represents a registered player
type User struct {
	Username string `json:"Username"`
	Password string `json:"PasswordHash"` // Store hashed password
	PlayerID string `json:"PlayerID"`
}
type Stats struct {
	HP        int `json:"HP"`
	Attack    int `json:"Attack"`
	Defense   int `json:"Defense"`
	Speed     int `json:"Speed"`
	SpAttack  int `json:"Sp_Attack"`
	SpDefense int `json:"Sp_Defense"`
}

type GenderRatio struct {
	MaleRatio   float64 `json:"MaleRatio"`
	FemaleRatio float64 `json:"FemaleRatio"`
}

type Profile struct {
	Height      float64     `json:"Height"`
	Weight      float64     `json:"Weight"`
	CatchRate   int         `json:"CatchRate"`
	GenderRatio GenderRatio `json:"GenderRatio"`
	EggGroup    string      `json:"EggGroup"`
	HatchSteps  int         `json:"HatchSteps"`
	Abilities   string      `json:"Abilities"`
}

type DamageCoefficient struct {
	Element     string  `json:"Element"`
	Coefficient float64 `json:"Coefficient"`
}

type Pokemon struct {
	Name               string              `json:"Name"`
	Elements           []string            `json:"Elements"`
	EV                 float64             `json:"EV"`
	Stats              Stats               `json:"Stats"`
	Profile            Profile             `json:"Profile"`
	DamageWhenAttacked []DamageCoefficient `json:"DamegeWhenAttacked"`
	EvolutionLevel     int                 `json:"EvolutionLevel"`
	NextEvolution      string              `json:"NextEvolution"`
	Moves              []string            `json:"Moves"`
	Experience         int                 `json:"Experience"`
	Level              int                 `json:"Level"`
}

type Player struct {
	ID       string    `json:"ID"`
	Name     string    `json:"Name"`
	Position [2]int    `json:"Position"`
	Caught   []Pokemon `json:"Caught"`
	AutoMode bool      `json:"AutoMode"`
}

type GameState struct {
	Players  map[string]*Player
	Pokemons map[[2]int]*Pokemon
	Mutex    sync.Mutex
	GridSize int
}

var gameState GameState

//-- Functions that handle the background logic of the game

func loadPokedex() []Pokemon {
	file, err := os.Open("../../pokedex/pokedex.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var pokedex []Pokemon
	if err := json.NewDecoder(file).Decode(&pokedex); err != nil {
		panic(err)
	}
	return pokedex
}

func spawnPokemons(pokedex []Pokemon, num int) {
	gameState.Mutex.Lock()
	defer gameState.Mutex.Unlock()

	for i := 0; i < num; i++ {
		x, y := rand.Intn(gameState.GridSize), rand.Intn(gameState.GridSize)
		pokemon := pokedex[rand.Intn(len(pokedex))]
		pokemon.Level = rand.Intn(100) + 1
		pokemon.EV = 0.5 + rand.Float64()*0.5
		gameState.Pokemons[[2]int{x, y}] = &pokemon
	}

	fmt.Println("[DEBUG] Total Pokémon Spawned:", len(gameState.Pokemons))
}

func initGameState(gridSize int) {
	gameState = GameState{
		Players:  make(map[string]*Player),
		Pokemons: make(map[[2]int]*Pokemon),
		GridSize: gridSize,
	}
}

const playerDataPath = "../../playerData"

func savePlayerData(player *Player) {
	// Ensure the directory exists
	if _, err := os.Stat(playerDataPath); os.IsNotExist(err) {
		err := os.MkdirAll(playerDataPath, os.ModePerm)
		if err != nil {
			fmt.Println("[ERROR] Error creating playerData directory:", err)
			return
		}
	}

	// Log player object
	fmt.Printf("[DEBUG] Player Object: %+v\n", *player)

	// Proceed to save
	filename := filepath.Join(playerDataPath, fmt.Sprintf("%s.json", player.ID))
	fmt.Println("[DEBUG] Saving player data at:", filename)

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("[ERROR] Error opening player data file:", err)
		return
	}
	defer file.Close()

	// Marshal player data into JSON
	jsonData, err := json.MarshalIndent(player, "", "  ")
	if err != nil {
		fmt.Println("[ERROR] Error marshalling player data:", err)
		return
	}
	fmt.Println("[DEBUG] JSON Data to be written:", string(jsonData))

	// Write JSON data to file
	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("[ERROR] Error writing to player data file:", err)
		return
	}

	fmt.Println("[DEBUG] Player data saved successfully:", filename)
}

func movePlayer(name string, direction string) {
	gameState.Mutex.Lock()
	defer gameState.Mutex.Unlock()

	player, exists := gameState.Players[name]
	if !exists {
		return
	}

	switch direction {
	case "up":
		if player.Position[1] > 0 {
			player.Position[1]--
		}
	case "down":
		if player.Position[1] < gameState.GridSize-1 {
			player.Position[1]++
		}
	case "left":
		if player.Position[0] > 0 {
			player.Position[0]--
		}
	case "right":
		if player.Position[0] < gameState.GridSize-1 {
			player.Position[0]++
		}
	}
	// Check to see the player capture any pokemon
	if pokemon, exists := gameState.Pokemons[player.Position]; exists {
		if len(player.Caught) < 200 {
			player.Caught = append(player.Caught, *pokemon)
			savePlayerData(player)
			delete(gameState.Pokemons, player.Position)
		}
	}
}

func toggleAutoMode(name string, enable bool) {
	gameState.Mutex.Lock()
	defer gameState.Mutex.Unlock()

	if player, exists := gameState.Players[name]; exists {
		player.AutoMode = enable
	}
}

func autoMovePlayers() {
	for {
		time.Sleep(1 * time.Second)
		gameState.Mutex.Lock()
		for _, player := range gameState.Players {
			if player.AutoMode {
				directions := []string{"up", "down", "left", "right"}
				movePlayer(player.Name, directions[rand.Intn(len(directions))])

			}
		}
		gameState.Mutex.Unlock()
	}
}

//---- This part of the program will handle http request between server and client

func handlePlayerJoin(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerID") // Passed after successful login

	if playerID == "" {
		w.Write([]byte("PlayerID is required to join"))
		return
	}

	gameState.Mutex.Lock()
	defer gameState.Mutex.Unlock()

	// Check if player already exists in the game state
	if _, exists := gameState.Players[playerID]; exists {
		w.Write([]byte("Player already joined"))
		return
	}

	// Build absolute path for player data using only PlayerID
	filename := filepath.Join(playerDataPath, fmt.Sprintf("%s.json", playerID))
	fmt.Println("[DEBUG] Looking for player data at:", filename)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("[ERROR] Error loading player data:", err)
		w.Write([]byte("Error loading player data"))
		return
	}
	defer file.Close()

	var player Player
	err = json.NewDecoder(file).Decode(&player)
	if err != nil {
		fmt.Println("[ERROR] Error decoding player data:", err)
		w.Write([]byte("Error decoding player data"))
		return
	}

	// Add player to game state using PlayerID as the key
	gameState.Players[playerID] = &player

	fmt.Println("[DEBUG] Player successfully joined:", player.Name, "ID:", player.ID)
	w.Write([]byte("Joined successfully"))
}

func handlePlayerLogin(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if username == "" || password == "" {
		w.Write([]byte("Username and password cannot be empty"))
		return
	}

	// Load users
	file, err := os.Open("users.json")
	if err != nil {
		w.Write([]byte("Error loading user data"))
		return
	}
	defer file.Close()
	usersFile := "users.json"
	absPath, _ := filepath.Abs(usersFile)
	fmt.Println("[DEBUG] users.json path:", absPath)

	var users []User
	json.NewDecoder(file).Decode(&users)

	// Find user
	for _, user := range users {
		if user.Username == username {
			// Verify password
			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
			if err == nil {
				w.Write([]byte(fmt.Sprintf("Login successful. PlayerID: %s", user.PlayerID)))
				return
			} else {
				w.Write([]byte("Invalid password"))
				return
			}
		}
	}

	w.Write([]byte("Username not found"))
}

func handlePlayerRegister(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if username == "" || password == "" {
		w.Write([]byte("Username and password cannot be empty"))
		return
	}

	// Load existing users
	usersFile := "users.json"
	file, err := os.OpenFile(usersFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("[ERROR] Error opening users file:", err)
		w.Write([]byte("Error opening users file"))
		return
	}
	defer file.Close()

	absPath, _ := filepath.Abs(usersFile)
	fmt.Println("[DEBUG] users.json path:", absPath)

	// Read existing users
	var users []User
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&users)
	if err != nil && err != io.EOF {
		fmt.Println("[ERROR] Error decoding users file:", err)
		w.Write([]byte("Error reading users data"))
		return
	}

	// Check if username already exists
	for _, user := range users {
		if user.Username == username {
			w.Write([]byte("Username already exists"))
			return
		}
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("[ERROR] Error hashing password:", err)
		w.Write([]byte("Error hashing password"))
		return
	}

	// Create a new player ID
	playerID := uuid.New().String()

	// Add new user
	newUser := User{
		Username: username,
		Password: string(hashedPassword),
		PlayerID: playerID,
	}
	users = append(users, newUser)

	// Clear file and write updated users
	err = file.Truncate(0) // Clear existing content
	if err != nil {
		fmt.Println("[ERROR] Error truncating users file:", err)
		w.Write([]byte("Error truncating users file"))
		return
	}

	_, err = file.Seek(0, 0) // Reset pointer to the start
	if err != nil {
		fmt.Println("[ERROR] Error seeking users file:", err)
		w.Write([]byte("Error seeking users file"))
		return
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON
	err = encoder.Encode(users)
	if err != nil {
		fmt.Println("[ERROR] Error writing to users file:", err)
		w.Write([]byte("Error saving user data"))
		return
	}

	// Ensure all data is written to disk
	err = file.Sync()
	if err != nil {
		fmt.Println("[ERROR] Error syncing users file:", err)
		w.Write([]byte("Error syncing users file"))
		return
	}

	fmt.Println("[DEBUG] New user registered successfully:", username)

	// Create player file with default data
	playerFile := filepath.Join(playerDataPath, fmt.Sprintf("%s.json", playerID))
	player := Player{
		ID:       playerID,
		Name:     username,
		Position: [2]int{0, 0}, // Default starting position
		Caught:   []Pokemon{},  // Empty Pokémon list
		AutoMode: false,
	}

	// Save initial player data to file
	playerFileHandle, err := os.OpenFile(playerFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("[ERROR] Error creating player data file:", err)
		w.Write([]byte("Error creating player data file"))
		return
	}
	defer playerFileHandle.Close()

	jsonData, err := json.MarshalIndent(player, "", "  ")
	if err != nil {
		fmt.Println("[ERROR] Error marshalling initial player data:", err)
		w.Write([]byte("Error marshalling player data"))
		return
	}

	_, err = playerFileHandle.Write(jsonData)
	if err != nil {
		fmt.Println("[ERROR] Error writing initial player data:", err)
		w.Write([]byte("Error writing player data"))
		return
	}

	fmt.Println("[DEBUG] Player data file created successfully:", playerFile)
	w.Write([]byte("Registration successful"))
}

func handlePlayerMove(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	direction := r.URL.Query().Get("direction")
	movePlayer(name, direction)
	w.Write([]byte("Moved"))
}

func handleAutoMode(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	enable := r.URL.Query().Get("enable") == "true"
	toggleAutoMode(name, enable)
	w.Write([]byte("Automode toggled"))

}
func handlePlayerSave(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	if name == "" {
		w.Write([]byte("Player name is required to save data. Example: /save?name=Ash"))
		return
	}

	gameState.Mutex.Lock()
	defer gameState.Mutex.Unlock()

	player, exists := gameState.Players[name]
	if !exists {
		w.Write([]byte("Player not found. Ensure you're joined in the game."))
		return
	}

	savePlayerData(player) // Save the player's data
	w.Write([]byte("Player data saved successfully!"))
}

func handleDebugGrid(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player")
	if playerID == "" {
		w.Write([]byte("PlayerID is required to display the grid around the player. Example: /debug/grid?player=<PlayerID>"))
		return
	}

	gameState.Mutex.Lock()
	defer gameState.Mutex.Unlock()

	// Debug: Check if the PlayerID exists
	player, exists := gameState.Players[playerID]
	if !exists {
		fmt.Println("[ERROR] Player not found in gameState.Players. PlayerID:", playerID)
		w.Write([]byte("Player not found. Please ensure you're joined in the game."))
		return
	}

	fmt.Println("[DEBUG] Found Player in gameState:", playerID, "Position:", player.Position)

	// Grid Size for Visualization
	viewSize := 50 // Display a 20x20 grid

	// Determine the player's center position
	centerX := player.Position[0]
	centerY := player.Position[1]

	// Start and End points for the view
	startX := max(0, centerX-viewSize/2)
	startY := max(0, centerY-viewSize/2)
	endX := min(gameState.GridSize, centerX+viewSize/2)
	endY := min(gameState.GridSize, centerY+viewSize/2)

	// Create a smaller grid view
	grid := make([][]string, endY-startY)
	for i := range grid {
		grid[i] = make([]string, endX-startX)
		for j := range grid[i] {
			grid[i][j] = "."
		}
	}

	// Add Pokémon to the visible grid
	for pos := range gameState.Pokemons {
		if pos[0] >= startX && pos[0] < endX && pos[1] >= startY && pos[1] < endY {
			x := pos[0] - startX
			y := pos[1] - startY
			grid[y][x] = "P"
		}
	}

	// Add Players to the visible grid
	for _, otherPlayer := range gameState.Players {
		if otherPlayer.Position[0] >= startX && otherPlayer.Position[0] < endX &&
			otherPlayer.Position[1] >= startY && otherPlayer.Position[1] < endY {
			x := otherPlayer.Position[0] - startX
			y := otherPlayer.Position[1] - startY
			grid[y][x] = "@"
		}
	}

	// Build and return the grid visualization
	var result string
	for _, row := range grid {
		result += fmt.Sprintf("%s\n", row)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(result))
}

// Helper Functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- Main function, here we will implement a place to call functions
// That we wrote to make a complete server

func main() {
	// Initialize Game State
	initGameState(1000)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("[ERROR] Could not get current working directory:", err)
	} else {
		fmt.Println("[DEBUG] Current working directory is:", cwd)
	}

	// Load Pokedex
	pokedex := loadPokedex()

	// Spawn Pokémon every minute
	go func() {
		for {
			spawnPokemons(pokedex, 100) // Spawn 50 Pokémon every minute
			time.Sleep(1 * time.Minute)
		}
	}()

	// Start Auto Movement for Players
	go autoMovePlayers()

	// HTTP Handlers
	http.HandleFunc("/register", handlePlayerRegister) // New
	http.HandleFunc("/login", handlePlayerLogin)       // New
	http.HandleFunc("/join", handlePlayerJoin)
	http.HandleFunc("/move", handlePlayerMove)
	http.HandleFunc("/automode", handleAutoMode)
	http.HandleFunc("/debug/grid", handleDebugGrid)
	http.HandleFunc("/save", handlePlayerSave)

	// Start Server
	fmt.Println("Server is running on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
