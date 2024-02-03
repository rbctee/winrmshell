package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/masterzen/winrm"
)

var (
	WarningLog *log.Logger
	InfoLog    *log.Logger
	ErrorLog   *log.Logger
)

func main() {

	InfoLog = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	WarningLog = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime)
	ErrorLog = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime)

	winrmServer := flag.String("server", "", "Server to connect to")
	winrmUsername := flag.String("username", "", "Username for authentication")
	winrmPassword := flag.String("password", "", "Password for authentication")
	winrmDomain := flag.String("domain", "", "Active Directory domain (DNS form)\nExample: example.local")
	winrmPort := flag.Int("port", 5985, "Port of WinRM service")
	useTLS := flag.Bool("tls", false, "Use TLS")
	verifyCert := flag.Bool("verify", false, "Verify TLS certificates")
	commandToExecute := flag.String("command", "", "Command to execute")

	flag.Parse()

	if *winrmServer == "" {
		InfoLog.Println("Missing server for WinRM connection")
		flag.Usage()
		return
	}

	if *winrmUsername == "" {
		InfoLog.Println("Missing username for authentication")
		flag.Usage()
		return
	}

	if *winrmPassword == "" {
		InfoLog.Println("Missing password for authentication")
		flag.Usage()
		return
	}

	if *commandToExecute == "" {
		InfoLog.Println("Missing command to execute")
		flag.Usage()
		return
	}

	if *winrmDomain == "" {
		WarningLog.Println("Missing domain, authentication will be performed LOCALLY!")
	}

	endpoint := winrm.NewEndpoint(*winrmServer, *winrmPort, *useTLS, *verifyCert, nil, nil, nil, 0)

	params := winrm.DefaultParameters
	params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }

	var authUsername string
	if *winrmDomain == "" {
		authUsername = *winrmUsername
	} else {
		authUsername = fmt.Sprintf("%s@%s", *winrmUsername, *winrmDomain)
	}
	client, err := winrm.NewClientWithParameters(endpoint, authUsername, *winrmPassword, params)
	if err != nil {
		ErrorLog.Printf("Error during authentication: %s\n", err)
		os.Exit(1)
	}

	shell, err := client.CreateShell()
	if err != nil {
		ErrorLog.Printf("Error creating shell: %s\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	commandArgs := strings.Split(*commandToExecute, " ")
	var command *winrm.Command
	if len(commandArgs) == 1 {
		command, err = shell.ExecuteWithContext(ctx, *commandToExecute)
	} else {
		command, err = shell.ExecuteWithContext(ctx, commandArgs[0], commandArgs[1:]...)
	}

	if err != nil {
		ErrorLog.Printf("Error during command execution: %s\n", err)
	}

	commandBytes := make([]byte, 4096)
	_, err = command.Stdout.Read(commandBytes)
	if err != nil {
		ErrorLog.Println("Failed to read bytes from command output")
	}

	InfoLog.Printf("Output of command:\n%s\n", commandBytes)
}
