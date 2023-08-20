package startgg

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"go.imnhan.com/gorts/players"
)

const STARTGG_URL = "https://api.start.gg/gql/alpha"

type GraphQL struct {
	Query     string   `json:"query"`
	Variables struct{} `json:"variables"`
}

type Inputs struct {
	Token string
	Slug  string
}

func LoadInputs(filepath string) Inputs {
	var result Inputs
	file, err := os.Open(filepath)
	if err != nil {
		return result
	}
	defer file.Close()

	s := bufio.NewScanner(file)
	s.Scan()
	result.Token = s.Text()
	s.Scan()
	result.Slug = s.Text()
	return result
}

func (c *Inputs) Write(filepath string) {
	blob := []byte(fmt.Sprintf("%s\n%s\n", c.Token, c.Slug))
	err := ioutil.WriteFile(filepath, blob, 0644)
	if err != nil {
		panic(err)
	}
}

// TODO: follow pagination
func FetchPlayers(i Inputs) ([]players.Player, error) {
	query := `
{
  tournament(slug: "%s") {
    participants(query: {page: %d, perPage: 500}) {
      nodes {
        entrants {
          event {
            slug
            name
          }
          team {
            name
          }
        }
        gamerTag
        prefix
        user {
          location {
            country
          }
        }
      }
    }
  }
}
`
	body, err := json.Marshal(GraphQL{
		Query:     fmt.Sprintf(query, i.Slug, 1),
		Variables: struct{}{},
	})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", STARTGG_URL, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("User-Agent", "GORTS/0.5")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+i.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch players: %w", err)
	}
	defer resp.Body.Close()

	respdata, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(">>>>", string(respdata[:50]))

	if resp.StatusCode != http.StatusOK {
		respJson := struct {
			Message string `json:"message"`
		}{}
		err = json.Unmarshal(respdata, &respJson)
		if err != nil {
			return nil, fmt.Errorf(
				"Unexpected %d response: %s", resp.StatusCode, respdata,
			)
		}
		return nil, errors.New(respJson.Message)
	}

	respJson := struct {
		Data struct {
			Tournament struct {
				Participants struct {
					Nodes []struct {
						// TODO: read team names from entrants too
						GamerTag string `json:"gamerTag"`
						Prefix   string `json:"prefix"`
						User     struct {
							Location struct {
								Country string `json:"country"`
							} `json:"location"`
						} `json:"user"`
					} `json:"nodes"`
				} `json:"participants"`
			} `json:"tournament"`
		} `json:"data"`
	}{}

	err = json.Unmarshal(respdata, &respJson)
	if err != nil {
		return nil, fmt.Errorf(
			"Unexpected %d response: %s", resp.StatusCode, respdata,
		)
	}

	participants := respJson.Data.Tournament.Participants.Nodes
	results := make([]players.Player, len(participants))
	for i, part := range participants {
		p := players.Player{}

		if part.Prefix == "" {
			p.Name = part.GamerTag
		} else {
			p.Name = fmt.Sprintf("%s %s", part.Prefix, part.GamerTag)
		}

		code, ok := countryNameToCode[part.User.Location.Country]
		if ok {
			p.Country = code
		} else if code != "" {
			fmt.Printf("*** Unknown country: %s\n", part.User.Location.Country)
		}

		results[i] = p
	}

	return results, nil
}


type StreamQueueVariables struct {
	TourneySlug string
}
type StreamQueueGraphQL struct {
	Query     string   `json:"query"`
	Variables struct {
		TourneySlug string `json:"tourneySlug"`
	} `json:"variables"`
}
func FetchLatestStreamQueue(i Inputs) (players.Player, players.Player, error) {
	query := `
	query StreamQueueOnTournament($tourneySlug: String!) {
		tournament(slug: $tourneySlug) {
		  id
		  streamQueue {
			stream {
			  streamSource
			  streamName
			}
			sets {
			  fullRoundText
			  slots {
				entrant {
				  participants {
					prefix
					gamerTag
					user {
						location {
						  country
						}
					  } 
				  }
				}
			  }
			}
		  }
		}
	  }	  
`
	body, err := json.Marshal(StreamQueueGraphQL{
		Query:     query,
		Variables: struct{TourneySlug string "json:\"tourneySlug\""}{
			TourneySlug: i.Slug,
		},
	})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", STARTGG_URL, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("User-Agent", "GORTS/0.5")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+i.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return players.Player{}, players.Player{}, fmt.Errorf("Error making API request: %w", err)
	}
	defer resp.Body.Close()

	respdata, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(">>>>", string(respdata[:50]))

	if resp.StatusCode != http.StatusOK {
		respJson := struct {
			Message string `json:"message"`
		}{}
		err = json.Unmarshal(respdata, &respJson)
		if err != nil {
			return players.Player{}, players.Player{}, fmt.Errorf(
				"Unexpected %d response: %s", resp.StatusCode, respdata,
			)
		}
		return players.Player{}, players.Player{}, errors.New(respJson.Message)
	}

	respJson := struct {
		Data struct {
			Tournament struct {
				StreamQueue []struct {
					Sets []struct {
						FullRoundText string `json:"fullRoundText"`
						Slots []struct {
							Entrant struct {
								Participants []struct {
									Prefix string `json:"prefix"`
									GamerTag string `json:"gamerTag"`
									User struct {
										Location struct {
											Country string `json:"country"`
										} `json:"location"`
									} `json:"user"`
								} `json:"participants"`
							} `json:"entrant"`
						} `json:"slots"`
					} `json:"sets"`
				} `json:"streamQueue"`
			} `json:"tournament"`
		} `json:"data"`
	}{}

	err = json.Unmarshal(respdata, &respJson)
	if err != nil {
		return players.Player{}, players.Player{}, fmt.Errorf(
			"Unexpected %d response: %s", resp.StatusCode, err.Error(),
		)
	}

	if (len(respJson.Data.Tournament.StreamQueue) > 0){
		playerOne := players.Player{
			Team: respJson.Data.Tournament.StreamQueue[0].Sets[0].Slots[0].Entrant.Participants[0].Prefix,
			Name: respJson.Data.Tournament.StreamQueue[0].Sets[0].Slots[0].Entrant.Participants[0].GamerTag,
		}
		playerOneCountry, ok := countryNameToCode[respJson.Data.Tournament.StreamQueue[0].Sets[0].Slots[0].Entrant.Participants[0].User.Location.Country]
		if ok {
			playerOne.Country = playerOneCountry
		}
		
		playerTwo := players.Player{
			Team: respJson.Data.Tournament.StreamQueue[0].Sets[0].Slots[1].Entrant.Participants[0].Prefix,
			Name: respJson.Data.Tournament.StreamQueue[0].Sets[0].Slots[1].Entrant.Participants[0].GamerTag,
			Country: respJson.Data.Tournament.StreamQueue[0].Sets[0].Slots[1].Entrant.Participants[0].User.Location.Country,
		}
		playerTwoCountry, ok := countryNameToCode[respJson.Data.Tournament.StreamQueue[0].Sets[0].Slots[1].Entrant.Participants[0].User.Location.Country]
		if ok {
			playerTwo.Country = playerTwoCountry
		}
	
		return playerOne, playerTwo, nil
	} else {
		return players.Player{}, players.Player{}, fmt.Errorf("No match found in stream queue")
	}
}
