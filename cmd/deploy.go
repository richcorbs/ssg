package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func deploy() {
	fmt.Println("Deploying...")

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		log.Fatalf("SSH_AUTH_SOCK environment variable is not set")
	}

	// Get the current user
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Failed to get current user: %v", err)
	}

	// Connect to the SSH agent
	conn, err := net.Dial("unix", sshAuthSock)
	if err != nil {
		log.Fatalf("failed to connect to SSH agent: %v", err)
	}

	// Authenticate with the agent
	sshAgent := agent.NewClient(conn)
	if sshAgent == nil {
		log.Fatalf("failed to authenticate with SSH agent")
	}

	keys, err := sshAgent.List()
	if err != nil {
		log.Fatalf("Failed to list keys from SSH agent: %v", err)
	}
	fmt.Println(len(keys))

	// Create an SSH client configuration
	config := &ssh.ClientConfig{
		User: currentUser.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(sshAgent.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // WARNING: Insecure, use proper host key verification in production
	}

	// Connect to the SSH server
	host := os.Getenv("DEPLOY_HOST")
	port := os.Getenv("DEPLOY_PORT")
	dir := os.Getenv("DEPLOY_DIR")
	sshClient, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		log.Fatalf("failed to connect to SSH server: %v", err)
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		log.Fatal("Failed to create SFTP client: ", err)
	}
	defer sftpClient.Close()

	localDir := "dist"
	remoteDir := dir

	err = deployDir(localDir, remoteDir, sftpClient)
	if err != nil {
		log.Fatal("Deploy failed:", err)
	}
}

func deployDir(localDir, remoteDir string, sftpClient *sftp.Client) error {
	_, err := sftpClient.Stat(remoteDir)
	if err == nil {
		// Remote directory exists, remove all files and subdirectories
		if err := deployRemoveAllFilesAndDirs(remoteDir, sftpClient); err != nil {
			return fmt.Errorf("failed to empty remote directory %s: %v", remoteDir, err)
		}
	} else if !os.IsNotExist(err) {
		// An error occurred while checking for the existence of the remote directory
		return fmt.Errorf("failed to check remote directory %s: %v", remoteDir, err)
	}
	// Open local directory
	localFiles, err := os.ReadDir(localDir)
	if err != nil {
		log.Fatalf("failed to read local directory: %v, %v", localDir, err)
	}

	for _, file := range localFiles {
		localPath := filepath.Join(localDir, file.Name())
		remotePath := filepath.Join(remoteDir, file.Name())

		if file.IsDir() {
			// Recurse directory
			err := deployDir(localPath, remotePath, sftpClient)
			if err != nil {
				return err
			}
		} else {
			// Ensure parent directory exists on the remote server
			parentDir := filepath.Dir(remotePath)
			if err := sftpClient.MkdirAll(parentDir); err != nil {
				return fmt.Errorf("failed to create remote directory %s: %v", parentDir, err)
			}

			// Transfer file
			srcFile, err := os.Open(localPath)
			if err != nil {
				return fmt.Errorf("failed to open local file %s: %v", file.Name(), err)
			}
			defer srcFile.Close()

			dstFile, err := sftpClient.Create(remotePath)
			if err != nil {
				return fmt.Errorf("failed to create remote file %s: %v", file.Name(), err)
			}
			defer dstFile.Close()

			_, err = io.Copy(dstFile, srcFile)
			if err != nil {
				return fmt.Errorf("failed to copy file %s: %v", file.Name(), err)
			}

			fmt.Printf("File %s deployed successfully\n", remoteDir+"/"+file.Name())
		}
	}
	return nil
}

func deployRemoveAllFilesAndDirs(dir string, sftpClient *sftp.Client) error {
	// Open directory
	files, err := sftpClient.ReadDir(dir)
	if err != nil {
		return err
	}

	// Remove each file and subdirectory
	for _, file := range files {
		filePath := filepath.Join(dir, file.Name())
		if file.IsDir() {
			// Recursively remove subdirectory
			if err := deployRemoveAllFilesAndDirs(filePath, sftpClient); err != nil {
				return err
			}
		} else {
			// Remove file
			if err := sftpClient.Remove(filePath); err != nil {
				return err
			}
		}
	}

	// Remove the directory itself
	if err := sftpClient.RemoveDirectory(dir); err != nil {
		return err
	}

	return nil
}
