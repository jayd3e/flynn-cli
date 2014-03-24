package main

import (
	"fmt"
	"log"
)

var cmdLogin = &Command{
	Run:   runLogin,
	Usage: "login",
	Short: "login to a Flynn instance",
	Long:  `Login to a Flynn instance by providing some required information`,
}

func runLogin(cmd *Command, args []string) {
	serv := server{}

	// Retrieve all the necessary information we need to represent an instance
	showPrompt("Git Host: ", &serv.GitHost)
	showPrompt("Api Url: ", &serv.ApiUrl)
	showPrompt("Api Key: ", &serv.ApiKey)
	showPrompt("Api TLS Pin: ", &serv.ApiTlsPin)

	config.Servers = append(config.Servers, &serv)
	writeConfig()
}

func showPrompt(prompt string, store *string) {
	fmt.Printf(prompt)
	_, err := fmt.Scanf("%s", store)
	if err != nil {
		log.Fatal("Couldn't retrieve user input:", err)
	}
}
