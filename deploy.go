package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// .env
// DEPLOY_TOKEN
// DEPLOY_PRODUCTION_DOMAIN
// DEPLOY_STAGING_DOMAIN
// set up variables
// if !DEPLOY_TOKEN
//   get token
// end
//
// if env == "staging" && domain
//   throw an error
// else if env == "production" && DEPLOY_PRODUCTION_DOMAIN && DOMAIN
//   throw an error
// else if env == "staging" && DEPLOY_STAGING_DOMAIN
//   deploy to staging
// else if env == "production" && DEPLOY_PRODUCTION_DOMAIN
//   deploy to production
// else if env == "production" && domain
//   write domain to DEPLOY_PRODUCTION_DOMAIN
//   deploy to production
// else if env == "staging" && !DEPLOY_STAGING_DOMAIN
//   get random domain
//   write random domain to DEPLOY_STAGING_DOMAIN
//   deploy to staging
// end

func deploy(domain string, env string) {
	var err error
	var deployDomain string
	if env == "production" {
		deployDomain = os.Getenv("DEPLOY_PRODUCTION_DOMAIN")
		if deployDomain != "" && domain != "" {
			// TODO: What really needs to happen here?
			log.Fatalf("You've already deployed this site to another domain: %v.", deployDomain)
		} else if deployDomain == "" && domain == "" {
			log.Fatal("You need to provide a domain or subdomain of sssg.live.")
		} else if deployDomain == "" && domain != "" {
			deployDomain = domain
		}
	} else if env == "staging" {
		deployDomain = os.Getenv("DEPLOY_STAGING_DOMAIN")
		if deployDomain == "" {
			deployDomain, err = getRandomDomain()
			if err != nil {
				fmt.Println("Error getting staging domain:", err)
				os.Exit(1)
			}
			err = os.Setenv("DEPLOY_STAGING_DOMAIN", deployDomain)
			if err != nil {
				fmt.Println("Error setting .env variable: DEPLOY_STAGING_DOMAIN", err)
				os.Exit(1)
			}
			writeEnv()
		}
	}

	fmt.Println("Deploying", env)
	localDir := "./dist/"

	token := os.Getenv("DEPLOY_TOKEN")
	if token == "" {
		token, err = register()
		if err != nil {
			fmt.Println("Error getting token:", err)
			os.Exit(1)
		}
		err = os.Setenv("DEPLOY_TOKEN", token)
		if err != nil {
			fmt.Println("Error setting .env variable DEPLOY_TOKEN:", err)
			os.Exit(1)
		}
		writeEnv()
	}

	err = registerDomain(deployDomain, token)
	if err != nil {
		fmt.Println("Error registering domain:", err)
		os.Exit(1)
	}

	// Create tar file
	tarFile := fmt.Sprintf("/tmp/%s.tar.gz", deployDomain)
	cmd := exec.Command("tar", "-czf", tarFile, "-C", localDir, ".")
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error creating content file:", tarFile, err)
		os.Exit(1)
	}

	// Send tar file to server
	// TODO: update the URL to use the production URL
	cmd = exec.Command("curl", "-F", "token="+token, "-F", "domain="+deployDomain, "-F", "file=@"+tarFile, "https://localhost/domain-upload")
	fmt.Println(cmd)
	err = cmd.Run()

	if err != nil {
		fmt.Printf("%s", err)
	}

	fmt.Println("Deployed:", deployDomain)
	// TODO: delete tarFile after deploy
}

func getRandomDomain() (domain string, error error) {
	fmt.Println("Getting random domain")
	cmd := exec.Command("curl", "https://localhost/domain-random")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("%s", err)
		return "", err
	}
	var parsedJson struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Domain  string `json:"domain"`
	}

	err = json.Unmarshal(out, &parsedJson)
	if err != nil {
		fmt.Printf("%s", err)
		return "", err
	}
	// fmt.Println(parsedJson.Domain)
	domain = parsedJson.Domain
	return domain, nil
}

func register() (token string, error error) {
	fmt.Println("Registering")
	cmd := exec.Command("curl", "-X", "POST", "https://localhost/register")
	out, err := cmd.Output()
	if err != nil {
		log.Println(string(out))
		fmt.Printf("%s\n", err)
		return "", err
	}
	var parsedJson struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Token   string `json:"token"`
	}

	err = json.Unmarshal(out, &parsedJson)
	if err != nil {
		fmt.Printf("%s", err)
		return "", err
	}
	fmt.Println(parsedJson.Token)
	token = parsedJson.Token
	return token, nil
}

func registerDomain(domain string, token string) (error error) {
	fmt.Println("Registering domain:", domain)
	cmd := exec.Command("curl", "-F", "token="+token, "-F", "domain="+domain, "https://localhost/domain-register")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("%s", err)
		return err
	}
	var parsedJson struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Domains string `json:"domains"`
	}

	err = json.Unmarshal(out, &parsedJson)
	if err != nil {
		fmt.Printf("%s", err)
		return err
	}
	return nil
}

func writeEnv() {
	var err error
	envContents := ""
	for _, e := range os.Environ() {
		if strings.Contains(e, "DEPLOY_") {
			envContents += e + "\n"
		}
	}
	err = os.WriteFile(".env", []byte(envContents), 0644)
	if err != nil {
		fmt.Println("Error writing environment variables to .env file:", err)
		os.Exit(1)
	}
}

// func deployOriginal() {
// 	fmt.Println("Deploying...")

// 	dir := os.Getenv("DEPLOY_DEST_DIR")
// 	host := os.Getenv("DEPLOY_HOST")
// 	port := os.Getenv("DEPLOY_PORT")
// 	user := os.Getenv("DEPLOY_USER")

// 	localDir := "./dist/"
// 	remoteDir := dir

// 	out, err := exec.Command("rsync", "-av", "--delete", "-e ssh -p "+port, localDir, user+"@"+host+":"+remoteDir).Output()

// 	if err != nil {
// 		fmt.Printf("%s", err)
// 	}

// 	fmt.Println("Rsync Successfully Executed")
// 	output := string(out[:])
// 	fmt.Println(output)
// }
