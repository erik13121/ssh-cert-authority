package main

import (
	"crypto/rand"
	"fmt"
	"github.com/cloudtools/ssh-cert-authority/client"
	"github.com/cloudtools/ssh-cert-authority/util"
	"github.com/codegangsta/cli"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io/ioutil"
	"net"
	"os"
	"time"
)

func trueOnError(err error) uint {
	if err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func requestCertFlags() []cli.Flag {
	validBeforeDur, _ := time.ParseDuration("2h")
	validAfterDur, _ := time.ParseDuration("-2m")
	home := os.Getenv("HOME")
	if home == "" {
		home = "/"
	}
	configPath := home + "/.ssh_ca/requester_config.json"

	return []cli.Flag{
		cli.StringFlag{
			Name:  "principals",
			Value: "ec2-user,ubuntu",
			Usage: "Valid usernames for login, comma separated (e.g. ec2-user,ubuntu)",
		},
		cli.StringFlag{
			Name:  "environment",
			Value: "",
			Usage: "An environment name (e.g. prod)",
		},
		cli.StringFlag{
			Name:  "config-file",
			Value: configPath,
			Usage: "Path to config.json",
		},
		cli.StringFlag{
			Name:  "reason",
			Value: "",
			Usage: "Your reason for needing this SSH certificate.",
		},
		cli.DurationFlag{
			Name:  "valid-after",
			Value: validAfterDur,
			Usage: "Relative time",
		},
		cli.DurationFlag{
			Name:  "valid-before",
			Value: validBeforeDur,
			Usage: "Relative time",
		},
	}
}

func requestCert(c *cli.Context) {
	config := make(map[string]ssh_ca_util.RequesterConfig)
	configPath := c.String("config-file")
	err := ssh_ca_util.LoadConfig(configPath, &config)
	if err != nil {
		fmt.Println("Load Config failed:", err)
		os.Exit(1)
	}

	reason := c.String("reason")
	if reason == "" {
		fmt.Println("Must give a reason for requesting this certificate.")
		os.Exit(1)
	}
	environment := c.String("environment")
	if len(config) > 1 && environment == "" {
		fmt.Println("You must tell me which environment to use.", len(config))
		os.Exit(1)
	}
	if len(config) == 1 && environment == "" {
		for environment = range config {
			// lame way of extracting first and only key from a map?
		}
	}

	_, ok := config[environment]
	if !ok {
		fmt.Printf("Environment '%s' not found in config file.", environment)
		os.Exit(1)
	}

	caRequest := ssh_ca_client.MakeRequest()
	caRequest.SetConfig(config[environment])
	failed := trueOnError(caRequest.SetEnvironment(environment))
	failed |= trueOnError(caRequest.SetReason(reason))
	failed |= trueOnError(caRequest.SetValidAfter(c.Duration("valid-after")))
	failed |= trueOnError(caRequest.SetValidBefore(c.Duration("valid-before")))
	failed |= trueOnError(caRequest.SetPrincipalsFromString(c.String("principals")))

	if failed == 1 {
		fmt.Println("One or more errors found. Aborting request.")
		os.Exit(1)
	}

	pubKeyContents, err := ioutil.ReadFile(config[environment].PublicKeyPath)
	if err != nil {
		fmt.Println("Trouble opening your public key file", config[environment].PublicKeyPath, err)
		os.Exit(1)
	}
	pubKey, pubKeyComment, _, _, err := ssh.ParseAuthorizedKey(pubKeyContents)
	if err != nil {
		fmt.Println("Trouble parsing your public key", err)
		os.Exit(1)
	}
	chosenKeyFingerprint := ssh_ca_util.MakeFingerprint(pubKey.Marshal())

	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		fmt.Println("Dial failed:", err)
		os.Exit(1)
	}
	sshAgent := agent.NewClient(conn)

	signers, err := sshAgent.Signers()
	var signer ssh.Signer
	signer = nil
	if err != nil {
		fmt.Println("No keys found in agent, can't sign request, bailing.")
		fmt.Println("ssh-add the private half of the key you want to use.")
		os.Exit(1)
	} else {
		for i := range signers {
			signerFingerprint := ssh_ca_util.MakeFingerprint(signers[i].PublicKey().Marshal())
			if signerFingerprint == chosenKeyFingerprint {
				signer = signers[i]
				break
			}
		}
	}
	if signer == nil {
		fmt.Println("ssh-add the private half of the key you want to use.")
		os.Exit(1)
	}
	caRequest.SetPublicKey(signer.PublicKey(), pubKeyComment)
	newCert, err := caRequest.EncodeAsCertificate()
	if err != nil {
		fmt.Println("Error encoding certificate request:", err)
		os.Exit(1)
	}
	err = newCert.SignCert(rand.Reader, signer)
	if err != nil {
		fmt.Println("Error signing:", err)
		os.Exit(1)
	}

	certRequest := newCert.Marshal()
	requestParameters := caRequest.BuildWebRequest(certRequest)
	requestID, err := caRequest.DoWebRequest(requestParameters)
	if err == nil {
		fmt.Printf("Cert request id: %s\n", requestID)
	} else {
		fmt.Println(err)
	}

}