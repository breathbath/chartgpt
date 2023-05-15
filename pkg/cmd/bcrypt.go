package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

var bcryptCmd = &cobra.Command{
	Use:   "bcrypt",
	Short: "Generates bcrypt hash from the prompted password imput",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("Enter password: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return errors.WithStack(err)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			return errors.WithStack(err)
		}

		fmt.Println(string(hashedPassword))

		return nil
	},
}

func initBcryptCmd() {
	rootCmd.AddCommand(bcryptCmd)
}
