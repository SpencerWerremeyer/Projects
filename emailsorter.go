package main

// removed to prevent errors
//  "reflect"
import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "net/url"
  "os"
  "os/user"
  "path/filepath"
  "encoding/base64"
  "strings"
  "regexp"
  "time"


  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/gmail/v1"
  "gopkg.in/ini.v1"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
  cacheFile, err := tokenCacheFile()
  if err != nil {
    log.Fatalf("Unable to get path to cached credential file. %v", err)
  }
  tok, err := tokenFromFile(cacheFile)
  if err != nil {
    tok = getTokenFromWeb(config)
    saveToken(cacheFile, tok)
  }
  return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
  authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
  fmt.Printf("Go to the following link in your browser then type the "+
    "authorization code: \n%v\n", authURL)

  var code string
  if _, err := fmt.Scan(&code); err != nil {
    log.Fatalf("Unable to read authorization code %v", err)
  }

  tok, err := config.Exchange(oauth2.NoContext, code)
  if err != nil {
    log.Fatalf("Unable to retrieve token from web %v", err)
  }
  return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
  usr, err := user.Current()
  if err != nil {
    return "", err
  }
  tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
  os.MkdirAll(tokenCacheDir, 0700)
  return filepath.Join(tokenCacheDir,
    url.QueryEscape("gmail-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
  openedFile, err := os.Open(file)
  if err != nil {
    return nil, err
  }
  token := &oauth2.Token{}
  err = json.NewDecoder(openedFile).Decode(token)
  defer openedFile.Close()
  return token, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
  fmt.Printf("Saving credential file to: %s\n", file)
  createdFile, err := os.Create(file)
  if err != nil {
    log.Fatalf("Unable to cache oauth token: %v", err)
  }
  defer createdFile.Close()
  json.NewEncoder(createdFile).Encode(token)
}

func writeToFile(file string, stringToFile string) {
  writer, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY , 0600)
  defer writer.Close()
  if err != nil {
    fmt.Println(err)
  }
  writer.WriteString(stringToFile)
  writer.Sync()
}

func main() {

  writeToFile("bin/test.txt", "test\n")

  // read the config file to a var
  configFile, err := ini.InsensitiveLoad("bin/test.ini")
  ctx := context.Background()

  fileToRead, err := ioutil.ReadFile("bin/client_secret.json")
  if err != nil {
    log.Fatalf("Unable to read client secret file: %v", err)
  }

  // If modifying these scopes, delete your previously saved credentials
  // at ~/.credentials/gmail-go-quickstart.json
  config, err := google.ConfigFromJSON(fileToRead, gmail.GmailComposeScope, gmail.GmailLabelsScope, gmail.GmailModifyScope)
  if err != nil {
    log.Fatalf("Unable to parse client secret file to config: %v", err)
  }
  client := getClient(ctx, config)

  srv, err := gmail.New(client)
  if err != nil {
    log.Fatalf("Unable to retrieve gmail Client %v", err)
  }

  user := "me"
  LabelList, err := srv.Users.Labels.List(user).Do()
  if err != nil {
    log.Fatalf("Unable to retrieve labels. %v", err)
  }

  // fmt.Println(reflect.TypeOf(LabelList.Labels))
  // fmt.Println(LabelList)



  // newArry := make([]*gmail.Label, int(1))

  // pls := srv.Users.Labels.Name("test")
  // testAppend := append(newArry, pls)
  // fmt.Println(testAppend)

  // testLabel := append(newArry, string())

  // newLabel, _ := srv.Users.Labels.Create(user, testLabel).Do()

  // fmt.Println(testLabel)
  // testLabel2, err := gmail.New(label)

  //TODO add the ability to create a label if one does not exist

  label := os.Args[1]
  OriginalLabel := ""
  CheckedLabel := ""
  OriginalLabelName := ""
  var systemName []string

  if (len(LabelList.Labels) > 0) {
    for _, currentLabel := range LabelList.Labels {
      if (strings.EqualFold(currentLabel.Name, label)){
        OriginalLabel = currentLabel.Id
        OriginalLabelName = currentLabel.Name
      }else if (strings.EqualFold(currentLabel.Name, "Checked_" + label)){
        CheckedLabel = currentLabel.Id
      }
    }
  } else {
    fmt.Print("No labels found.")
  }

    keyNames := configFile.Section(label).KeyStrings()
    keyValues := []string{}

    for _, name := range keyNames{
      keyValues = append(keyValues, configFile.Section(label).Key(name).String())
    }

    //get a list of messages that meet the terms of the search
    response, err := srv.Users.Messages.List(user).Q("label:"+ OriginalLabelName).Do()

    if err != nil {
      log.Fatalf("Unable to find messages with the " + label + " label.")
    }


    var idList []string
    // check if message exists and add it to a list
    if (len(response.Messages) > 0) {
      for _, currentMessage := range response.Messages {
        idList = append(idList, currentMessage.Id)
      }
    }

    if (len(idList) > 0) {
      for _, message := range idList{
        msg, err := srv.Users.Messages.Get(user, string(message)).Do()
        if err != nil {
          log.Fatalf("Unable to retrieve message")
        }

        labels := msg.LabelIds

        // fmt.Println(CheckedLabelName)
        //check the list of messages for certain labels
        if !strings.Contains(strings.Join(labels, " "), CheckedLabel) && strings.Contains(strings.Join(labels, " "), OriginalLabel){

          var decodedBody []byte
          var stringMessage string
          var timeFMT string = ""


          //loop through a list of key names
          for _, regexValue := range keyValues {
            // fmt.Println(regexValue)

            //if Parts was found in the confing file decode based on constents
            if regexValue == "Parts"{
                decodedBody, _ = base64.StdEncoding.DecodeString(msg.Payload.Parts[0].Body.Data)
            } else if regexValue == "Payload"{
                decodedBody, _ = base64.StdEncoding.DecodeString(string(msg.Payload.Body.Data))
            }
            // check if decodedBody was assigned
            if len(decodedBody) != 0{
              stringMessage = string(decodedBody)
            }
            //compile the regexValue
            compiledVar := regexp.MustCompile(regexValue)

            //TODO use this to add the system name to a string that contains the time


            //check if there is a time fmt and fimd times
            if timeFMT != ""{

              foundDate := strings.Join(compiledVar.FindAllString(stringMessage, 1), " ")
              timeVar, err := time.Parse(timeFMT, foundDate)
              if err != nil{
                fmt.Println("timeVar not set")
              }

              fmt.Println(timeVar)
              timeFMT = ""

            }else if strings.Contains(regexValue, "2006"){
              timeSplit := strings.Split(regexValue, "/")
              // fmt.Println(timeSplit[0])
              if timeSplit[0] == "2006" {
                upsTime := regexValue
              }
              timeFMT = regexValue
            }

            stringToPrint := compiledVar.FindAllString(stringMessage, 1)


            if strings.Contains(regexValue, "System Name: ") {
              systemName = stringToPrint
            }

            if len(stringToPrint) != 0{
              fmt.Println(stringToPrint)
            }
            //Adds "checked" label from the current message
            label := []string{CheckedLabel}
            mod := &gmail.ModifyMessageRequest{AddLabelIds: label}
            srv.Users.Messages.Modify(user, message, mod).Do()
          }
          if len(systemName) != 0{
            fmt.Println(systemName)
          }
          fmt.Println(upsTime)
          fmt.Println()


          //TODO create log file of when ups starts and stop time ups name ex (c1, r1 etc)
          //don't support dayliight savings time
          //find a way to get date field from email
          //create new lable if one dosn't already exist

          //create duplicate of the function in sandbox with a restricted user


        }
      }
    }

}
