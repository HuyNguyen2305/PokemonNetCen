package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Define host and port for easy configuration
const (
	Host = "http://192.168.1.15" // Change this to your desired host IP
	Port = "8080"                // Change this to your desired port
)

// Send HTTP Request Helper
func sendRequest(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return ""
	}

	return string(body)
}

// Register a New Account
func register() string {
	var username, password string
	fmt.Print("Enter username for registration: ")
	fmt.Scanln(&username)
	fmt.Print("Enter password for registration: ")
	fmt.Scanln(&password)

	response := sendRequest(fmt.Sprintf("%s:%s/register?username=%s&password=%s", Host, Port, username, password))
	fmt.Println(response)

	if strings.Contains(response, "successful") {
		return username
	}
	return ""
}

// Login to an Existing Account
func login() string {
	var username, password string
	fmt.Print("Enter username for login: ")
	fmt.Scanln(&username)
	fmt.Print("Enter password for login: ")
	fmt.Scanln(&password)

	response := sendRequest(fmt.Sprintf("%s:%s/login?username=%s&password=%s", Host, Port, username, password))
	fmt.Println(response)

	if strings.Contains(response, "PlayerID:") {
		parts := strings.Split(response, "PlayerID: ")
		if len(parts) > 1 {
			playerID := strings.TrimSpace(parts[1])
			return playerID
		}
	}
	return ""
}

// Display the Grid
func showGrid(playerID string) {
	url := fmt.Sprintf("%s:%s/debug/grid?player=%s", Host, Port, playerID)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching grid:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading grid:", err)
		return
	}

	fmt.Println(string(body))
}

// Main Game Loop
func main() {
	fmt.Println("Welcome to PokéCat!")

	var choice int
	var playerID string

	// Authentication Loop
	for {
		fmt.Println("\n1. Register\n2. Login\n3. Quit")
		fmt.Print("Choose an option: ")
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			username := register()
			if username != "" {
				fmt.Println("Registration successful! Please login now.")
			}
		case 2:
			playerID = login()
			if playerID != "" {
				fmt.Println("Login successful! PlayerID:", playerID)
				// Explicitly call join after login
				sendRequest(fmt.Sprintf("%s:%s/join?playerID=%s", Host, Port, playerID))
				goto GameLoop
			} else {
				fmt.Println("Login failed, please try again.")
			}

		case 3:
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice, please select again.")
		}
	}

GameLoop:
	fmt.Println("Joined the game successfully!")

	for {
		var input string
		fmt.Println("\nEnter command (w/a/s/d for move, auto on/off, grid, save, quit):")
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "w":
			sendRequest(fmt.Sprintf("%s:%s/move?name=%s&direction=up", Host, Port, playerID))
		case "a":
			sendRequest(fmt.Sprintf("%s:%s/move?name=%s&direction=left", Host, Port, playerID))
		case "s":
			sendRequest(fmt.Sprintf("%s:%s/move?name=%s&direction=down", Host, Port, playerID))
		case "d":
			sendRequest(fmt.Sprintf("%s:%s/move?name=%s&direction=right", Host, Port, playerID))
		case "auto on":
			sendRequest(fmt.Sprintf("%s:%s/automode?name=%s&enable=true", Host, Port, playerID))
		case "auto off":
			sendRequest(fmt.Sprintf("%s:%s/automode?name=%s&enable=false", Host, Port, playerID))
		case "grid":
			showGrid(playerID)
		case "save":
			sendRequest(fmt.Sprintf("%s:%s/save?name=%s", Host, Port, playerID))
			fmt.Println("Game state saved.")
		case "quit":
			fmt.Println("Exiting the game.")
			return
		default:
			fmt.Println("Unknown command, please try again.")
		}
	}
}
