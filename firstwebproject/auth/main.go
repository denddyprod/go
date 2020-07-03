package main

import (
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string
	Password string
	Email    string
	GToken   string
}

type Credentials struct {
	Password      string
	Username      string
	Authenticated bool
}

type HTMLData struct {
	SiteKey   string
	RegErrStr string
	LogErrStr string
}

var templates *template.Template
var db *sql.DB
var err error
var myEnv map[string]string
var dataHTML HTMLData
var store *sessions.CookieStore

func (h *HTMLData) resetData() {
	if len(h.LogErrStr) > 0 {
		h.LogErrStr = ""
	}
	if len(h.RegErrStr) > 0 {
		h.RegErrStr = ""
	}
}

func connectDB() *sql.DB {
	username := myEnv["databaseUser"]
	password := myEnv["databasePassword"]
	databaseName := myEnv["databaseName"]
	databaseHost := myEnv["databaseHost"]
	databaseType := myEnv["databaseType"]

	dbURI := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		username, password, databaseHost, databaseName)

	db, err = sql.Open(databaseType, dbURI)
	if err != nil {
		panic(err.Error())
	}

	return db
}

func (user User) validateFields() error {
	return validation.ValidateStruct(&user,
		validation.Field(&user.Username, validation.Required, validation.Length(3, 20)),
		validation.Field(&user.Password, validation.Required, validation.Length(6, 35)),
		validation.Field(&user.Email, validation.Required, is.Email),
	)
}

func verifyCaptcha(user User) error {
	secretKey := myEnv["secretKey"]

	urlStr := "https://www.google.com/recaptcha/api/siteverify"

	urlParameters := url.Values{
		"secret":   {secretKey},
		"response": {user.GToken},
	}
	resp, err := http.PostForm(urlStr, urlParameters)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		print(err)
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	if result["success"] != true {
		err = errors.New("Verification: You look like a bot," +
			"please contact us for more information!")
		return err
	}

	return nil
}

func validateSignUp(user User, pwdConfirm string) (bool, error) {

	err := user.validateFields()
	if err != nil {
		return false, err
	}

	err = validation.Validate(pwdConfirm,
		validation.Required,
		validation.Length(6, 35),
	)
	if err != nil {
		errString := "Confirmation Password: " + err.Error()
		err = errors.New(errString)
		return false, err
	}

	if pwdConfirm != user.Password {
		err = errors.New("Password: Pasword and " +
			"Confirmation Password must be same")
		return false, err
	}

	err = verifyCaptcha(user)
	if err != nil {
		return false, err
	}

	return true, nil
}

func signUpHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	user.Username = r.FormValue("username")
	user.Email = r.FormValue("email")
	user.Password = r.FormValue("password")
	user.GToken = r.FormValue("g-token")
	pwdConfirm := r.FormValue("confirm")

	status, err := validateSignUp(user, pwdConfirm)
	if status != true {
		dataHTML.RegErrStr = err.Error()
		fmt.Println(err)
		http.Redirect(w, r, "/", 301)
	} else {
		var res string
		err := db.QueryRow("SELECT username FROM users WHERE username=?", user.Username).Scan(&res)

		switch {
		case err == sql.ErrNoRows:
			hashPwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Server error 1, unable to create your account.", 500)
				return
			}

			_, err = db.Exec("INSERT INTO users(username, password, email) VALUES(?, ?, ?)", user.Username, hashPwd, user.Email)
			if err != nil {
				http.Error(w, "Server error 2, unable to create your account.", 500)
				return
			}

			w.Write([]byte("User created!"))
			return
		case err != nil:
			http.Error(w, "Server error 3, unable to create your account.", 500)
			return
		default:
			dataHTML.RegErrStr = "This username already exists! "
			http.Redirect(w, r, "/", 301)
		}
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	creds.Username = r.FormValue("login_username")
	creds.Password = r.FormValue("login_password")
	creds.Authenticated = false

	err1 := validation.Validate(creds.Username,
		validation.Required,
	)
	err2 := validation.Validate(creds.Password,
		validation.Required,
	)
	if err1 != nil || err2 != nil {
		fmt.Print("[Empty Data] All fields must be filled!\n")
	}

	var databaseUsername string
	var databasePassword string

	err := db.QueryRow("SELECT username, password FROM users WHERE username=?", creds.Username).Scan(&databaseUsername, &databasePassword)
	if err != nil {
		errStr := errors.New("Incorrect Username or Password! Please, try again!")
		dataHTML.LogErrStr = errStr.Error()
		fmt.Println(err)
		http.Redirect(w, r, "/", 301)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(databasePassword), []byte(creds.Password))
	if err != nil {
		errStr := errors.New("Incorrect Username or Password! Please, try again!")
		dataHTML.LogErrStr = errStr.Error()
		fmt.Println(err)
		http.Redirect(w, r, "/", 301)
		return
	}

	session, err := store.Get(r, "session-login")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	creds.Authenticated = true
	session.Values["user"] = creds
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Hello, " + databaseUsername))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", dataHTML)
	dataHTML.resetData()
}

func getUser(s *sessions.Session) Credentials {
	e := s.Values["user"]
	var cred = Credentials{}
	cred, status := e.(Credentials)
	if !status {
		return Credentials{Authenticated: false}
	}

	return cred
}

func secretHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-login")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	authUser := getUser(session)

	if status := authUser.Authenticated; !status {
		session.AddFlash("NO ACCESS!")
		return
	}

	w.Write([]byte("Hello, " + authUser.Username))
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-login")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["user"] = Credentials{}
	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/", http.StatusFound)
}

func startServer() *mux.Router {
	serverPort := myEnv["PORT"]
	templates = template.Must(template.ParseGlob("templates/*.html"))

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/signup", signUpHandler)
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/secret", secretHandler)
	r.HandleFunc("/logout", logoutHandler)
	r.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("templates/css/"))))

	port := fmt.Sprintf(":%s", serverPort)

	fmt.Println("Well done! Server started")
	http.ListenAndServe(port, r)

	return r
}

func initSessions() *sessions.CookieStore {
	authKey := securecookie.GenerateRandomKey(64)
	encryptionKey := securecookie.GenerateRandomKey(32)

	store = sessions.NewCookieStore(
		authKey,
		encryptionKey,
	)

	store.Options = &sessions.Options{
		MaxAge:   60 * 15,
		HttpOnly: true,
	}

	gob.Register(Credentials{})

	return store
}

func main() {
	myEnv, err = godotenv.Read()
	dataHTML.SiteKey = myEnv["siteKey"]
	if err != nil {
		log.Fatalf("Error loading .env file %s", err)
	}

	store = initSessions()
	db = connectDB()
	defer db.Close()
	r := startServer()

	_ = r

}
