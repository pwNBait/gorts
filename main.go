package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"go.imnhan.com/gorts/ipc"
	"go.imnhan.com/gorts/players"
	"go.imnhan.com/gorts/startgg"
)

const WebPort = "1337"
const WebDir = "web"
const ScoreboardFile = WebDir + "/state.json"
const BracketFile = WebDir + "/bracket.json"
const PlayersFile = "players.csv"
const StartggFile = "creds-startgg"

func main() {
	// No need to wait on the http server,
	// just let it die when the GUI is closed.
	go func() {
		println("Serving scoreboard at http://localhost:" + WebPort)
		fs := http.FileServer(http.Dir(WebDir))
		http.Handle("/", fs)
		err := http.ListenAndServe("127.0.0.1:"+WebPort, nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	tclPathPtr := flag.String("tcl", DefaultTclPath, "Path to tclsh executable")
	flag.Parse()

	startGUI(*tclPathPtr)
}

func startGUI(tclPath string) {
	cmd := exec.Command(tclPath, "-encoding", "utf-8")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	go func() {
		errscanner := bufio.NewScanner(stderr)
		for errscanner.Scan() {
			errtext := errscanner.Text()
			fmt.Printf("XXX %s\n", errtext)
		}
	}()

	fmt.Fprintln(stdin, `source -encoding "utf-8" tcl/main.tcl`)
	println("Loaded main tcl script.")

	allplayers := players.FromFile(PlayersFile)
	scoreboard := initScoreboard()
	startggInputs := startgg.LoadInputs(StartggFile)

	fmt.Fprintln(stdin, "initialize")

	respond := func(values ...string) {
		ipc.Respond(stdin, values)
	}

	for req := range ipc.IncomingRequests(stdout) {
		switch req.Method {

		case "forcefocus":
			err := forceFocus(req.Args[0])
			if err != nil {
				fmt.Printf("forcefocus: %s\n", err)
			}
			respond("ok")

		case "getstartgg":
			respond(startggInputs.Token, startggInputs.Slug)

		case "getwebport":
			respond(WebPort)

		case "getcountrycodes":
			respond(startgg.CountryCodes...)

		case "getscoreboard":
			// TODO: there must be a more... civilized way.
			respond(
				scoreboard.Description,
				scoreboard.Subtitle,
				scoreboard.P1name,
				scoreboard.P1country,
				strconv.Itoa(scoreboard.P1score),
				scoreboard.P1team,
				scoreboard.P2name,
				scoreboard.P2country,
				strconv.Itoa(scoreboard.P2score),
				scoreboard.P2team,
			)

		case "applyscoreboard":
			scoreboard.Description = req.Args[0]
			scoreboard.Subtitle = req.Args[1]
			scoreboard.P1name = req.Args[2]
			scoreboard.P1country = req.Args[3]
			scoreboard.P1score, _ = strconv.Atoi(req.Args[4])
			scoreboard.P1team = req.Args[5]
			scoreboard.P2name = req.Args[6]
			scoreboard.P2country = req.Args[7]
			scoreboard.P2score, _ = strconv.Atoi(req.Args[8])
			scoreboard.P2team = req.Args[9]
			scoreboard.C1Title = req.Args[10]
			scoreboard.C1Subtitle = req.Args[11]
			scoreboard.C2Title = req.Args[12]
			scoreboard.C2Subtitle = req.Args[13]
			scoreboard.Write()
			respond()

		case "searchplayers":
			query := req.Args[0]
			var names []string

			if query == "" {
				for _, p := range allplayers {
					names = append(names, p.Name)
				}
				respond(names...)
				break
			}

			for _, p := range allplayers {
				if p.MatchesName(query) {
					names = append(names, p.Name)
				}
			}
			respond(names...)

		case "fetchplayers":
			startggInputs.Token = req.Args[0]
			startggInputs.Slug = req.Args[1]
			ps, err := startgg.FetchPlayers(startggInputs)
			fmt.Fprintln(stdin, "fetchplayers__resp")
			if err != nil {
				respond("err", fmt.Sprintf("Error: %s", err))
				break
			}
			allplayers = ps
			// TODO: show write errors to user instead of ignoring
			startggInputs.Write(StartggFile)
			players.Write(PlayersFile, allplayers)
			respond("ok", fmt.Sprintf("Successfully fetched %d players.", len(allplayers)))

		case "fetchlateststreamqueue":
			startggInputs.Token = req.Args[0]
			startggInputs.Slug = req.Args[1]
			playerOne, playerTwo, err := startgg.FetchLatestStreamQueue(startggInputs)
			fmt.Fprintln(stdin, "getstreamqueue__resp")
			if err != nil {
				respond("err", fmt.Sprintf("Error: %s", err))
				break
			}
			respond("ok",
			 "Successfully fetched stream match.",
			 playerOne.Name,
			 playerOne.Country,
			 "0",
			 playerOne.Team,
			 playerTwo.Name,
			 playerTwo.Country,
			 "0",
			 playerTwo.Team)
		case "fetchbracket":
			startggInputs.Token = req.Args[0]
			startggInputs.PhaseGroupId = req.Args[1]
			bracket, err := startgg.FetchBracket(startggInputs)
			fmt.Fprintln(stdin, "getbracket__resp")
			if err != nil {
				respond("err", fmt.Sprintf("Error: %s", err))
				break
			}
			err = WriteBracket(bracket)
			if err != nil {
				respond("err", fmt.Sprintf("Error: %s", err))
				break
			}
			respond("ok",
				"Successfully fetched bracket.")


		case "clearstartgg":
			startggInputs = startgg.Inputs{}
			startggInputs.Write(StartggFile)

		case "getplayercountry":
			playerName := req.Args[0]
			var country string
			for _, p := range allplayers {
				if p.Name == playerName {
					country = p.Country
					break
				}
			}
			respond(country)
		}
	}

	println("Tcl process terminated.")
}

type Scoreboard struct {
	Description string `json:"description"`
	Subtitle    string `json:"subtitle"`
	P1name      string `json:"p1name"`
	P1country   string `json:"p1country"`
	P1score     int    `json:"p1score"`
	P1team      string `json:"p1team"`
	P2name      string `json:"p2name"`
	P2country   string `json:"p2country"`
	P2score     int    `json:"p2score"`
	P2team      string `json:"p2team"`
	C1Title      string `json:"c1title"`
	C1Subtitle      string `json:"c1subtitle"`
	C2Title      string `json:"c2title"`
	C2Subtitle      string `json:"c2subtitle"`
}

func initScoreboard() Scoreboard {
	var scoreboard Scoreboard
	file, err := os.Open(ScoreboardFile)
	if err == nil {
		defer file.Close()
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		json.Unmarshal(bytes, &scoreboard)
	}
	return scoreboard
}

func (s *Scoreboard) Write() {
	blob, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(ScoreboardFile, blob, 0644)
	if err != nil {
		panic(err)
	}
}

type BracketJson struct {
	Ltop82p1 string `json:"ltop82p1"`
	Ltop82p1s string `json:"ltop82p1s"`
	Ltop82p2 string `json:"ltop82p2"`
	Ltop82p2s string `json:"ltop82p2s"`
	Ltop81p1 string `json:"ltop81p1"`
	Ltop81p1s string `json:"ltop81p1s"`
	Ltop81p2 string `json:"ltop81p2"`
	Ltop81p2s string `json:"ltop81p2s"`
	Lq2p1 string `json:"lq2p1"`
	Lq2p1s string `json:"lq2p1s"`
	Lq2p2 string `json:"lq2p2"`
	Lq2p2s string `json:"lq2p2s"`
	Lq1p1 string `json:"lq1p1"`
	Lq1p1s string `json:"lq1p1s"`
	Lq1p2 string `json:"lq1p2"`
	Lq1p2s string `json:"lq1p2s"`
	Lsp1 string `json:"lsp1"`
	Lsp1s string `json:"lsp1s"`
	Lsp2 string `json:"lsp2"`
	Lsp2s string `json:"lsp2s"`
	Lfp1 string `json:"lfp1"`
	Lfp1s string `json:"lfp1s"`
	Lfp2 string `json:"lfp2"`
	Lfp2s string `json:"lfp2s"`
	Ws2p1 string `json:"ws2p1"`
	Ws2p1s string `json:"ws2p1s"`
	Ws2p2 string `json:"ws2p2"`
	Ws2p2s string `json:"ws2p2s"`
	Ws1p1 string `json:"ws1p1"`
	Ws1p1s string `json:"ws1p1s"`
	Ws1p2 string `json:"ws1p2"`
	Ws1p2s string `json:"ws1p2s"`
	Wfp1 string `json:"wfp1"`
	Wfp1s string `json:"wfp1s"`
	Wfp2 string `json:"wfp2"`
	Wfp2s string `json:"wfp2s"`
	Gfp1 string `json:"gfp1"`
	Gfp1s string `json:"gfp1s"`
	Gfp2 string `json:"gfp2"`
	Gfp2s string `json:"gfp2s"`
}

//TODO: Allow arbitrary sizes for phase group
func WriteBracket(bracket startgg.Bracket) error {
	//matches := []string{ "ltop82", "ltop81", "lq2", "lq1", "ls", "lf", "ws2", "ws1", "wf", "gf"}
	if len(bracket) < 10 {
		return fmt.Errorf("Too few matches retrieved from phase. %d matches", len(bracket)) 
	}
	bracketJson := BracketJson{
		Lfp1 : bracket[0].PlayerOne.Name,
		Lfp1s : bracket[0].PlayerOne.Score,
		Lfp2 : bracket[0].PlayerTwo.Name,
		Lfp2s : bracket[0].PlayerTwo.Score,
		Lsp1 : bracket[1].PlayerOne.Name,
		Lsp1s : bracket[1].PlayerOne.Score,
		Lsp2 : bracket[1].PlayerTwo.Name,
		Lsp2s : bracket[1].PlayerTwo.Score,
		Lq1p1 : bracket[2].PlayerOne.Name,
		Lq1p1s : bracket[2].PlayerOne.Score,
		Lq1p2 : bracket[2].PlayerTwo.Name,
		Lq1p2s : bracket[2].PlayerTwo.Score,
		Lq2p1 : bracket[3].PlayerOne.Name,
		Lq2p1s : bracket[3].PlayerOne.Score,
		Lq2p2 : bracket[3].PlayerTwo.Name,
		Lq2p2s : bracket[3].PlayerTwo.Score,
		Ltop81p1 : bracket[4].PlayerOne.Name,
		Ltop81p1s : bracket[4].PlayerOne.Score,
		Ltop81p2 : bracket[4].PlayerTwo.Name,
		Ltop81p2s : bracket[4].PlayerTwo.Score,
		Ltop82p1 : bracket[5].PlayerOne.Name,
		Ltop82p1s : bracket[5].PlayerOne.Score,
		Ltop82p2 : bracket[5].PlayerTwo.Name,
		Ltop82p2s : bracket[5].PlayerTwo.Score,
		Ws2p1 : bracket[6].PlayerOne.Name,
		Ws2p1s : bracket[6].PlayerOne.Score,
		Ws2p2 : bracket[6].PlayerTwo.Name,
		Ws2p2s : bracket[6].PlayerTwo.Score,
		Ws1p1 : bracket[7].PlayerOne.Name,
		Ws1p1s : bracket[7].PlayerOne.Score,
		Ws1p2 : bracket[7].PlayerTwo.Name,
		Ws1p2s : bracket[7].PlayerTwo.Score,
		Wfp1 : bracket[8].PlayerOne.Name,
		Wfp1s : bracket[8].PlayerOne.Score,
		Wfp2 : bracket[8].PlayerTwo.Name,
		Wfp2s : bracket[8].PlayerTwo.Score,
		Gfp1 : bracket[9].PlayerOne.Name,
		Gfp1s : bracket[9].PlayerOne.Score,
		Gfp2 : bracket[9].PlayerTwo.Name,
		Gfp2s : bracket[9].PlayerTwo.Score,
	}

	blob, err := json.MarshalIndent(bracketJson, "", "    ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(BracketFile, blob, 0644)
	if err != nil {
		panic(err)
	}
	return nil
}