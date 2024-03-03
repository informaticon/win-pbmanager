package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	pwsh("codium", "a", "b")
}

func diffWithPwsh(command, myDir, otherDir string) {
	cmd := exec.Command("powershell", "-nologo", "-noprofile")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdin.Close()
	fmt.Fprintln(stdin, fmt.Sprintf("%s '%s' '%s' --wait --new-window", command, myDir, otherDir))
	fmt.Fprintln(stdin, "exit")
	cmd.Env = append(os.Environ(), "COMPARE_FOLDERS=DIFF")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)
}
