package registry

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"path"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/okta"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ID_LEN = 12
)

type User struct {
	Id       string `gorm:"primaryKey"`
	Created  int64  `gorm:"autoCreateTime"`
	Updated  int64  `gorm:"autoUpdateTime"`
	Email    string `gorm:"unique"`
	Password []byte
	Admin    bool

	Sources     []Source     `gorm:"many2many:users_sources"`
	Permissions []Permission `gorm:"foreignKey:UserId;references:Id"`
}

var (
	SOURCE_TYPE_INFRA = "infra"
	SOURCE_TYPE_OKTA  = "okta"
)

type Source struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Type    string `yaml:"type"`

	OktaDomain       string `gorm:"unique"`
	OktaClientId     string
	OktaClientSecret string
	OktaApiToken     string

	Users []User `gorm:"many2many:users_sources"`

	FromConfig bool
}

var (
	DESTINATION_TYPE_KUBERNERNETES = "kubernetes"
)

type Destination struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Name    string `gorm:"unique"`
	Type    string

	KubernetesCa        string
	KubernetesEndpoint  string
	KubernetesNamespace string
}

type Permission struct {
	Id            string `gorm:"primaryKey"`
	Created       int64  `gorm:"autoCreateTime"`
	Updated       int64  `gorm:"autoUpdateTime"`
	Role          string
	UserId        string
	DestinationId string
	User          User        `gorm:"foreignKey:UserId;references:Id"`
	Destination   Destination `gorm:"foreignKey:DestinationId;references:Id"`

	FromConfig  bool
	FromDefault bool
}

type Settings struct {
	Id         string `gorm:"primaryKey"`
	Created    int64  `gorm:"autoCreateTime"`
	Updated    int64  `gorm:"autoUpdateTime"`
	PrivateJWK []byte
	PublicJWK  []byte
}

var (
	TOKEN_SECRET_LEN = 24
	TOKEN_LEN        = ID_LEN + TOKEN_SECRET_LEN
)

type Token struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Expires int64
	Secret  []byte

	UserId string
	User   User `gorm:"foreignKey:UserId;references:Id;"`
}

var (
	API_KEY_LEN = 24
)

type ApiKey struct {
	Id      string `gorm:"primaryKey"`
	Created int64  `gorm:"autoCreateTime"`
	Updated int64  `gorm:"autoUpdateTime"`
	Name    string `gorm:"unique"`
	Key     string
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.Id == "" {
		u.Id = generate.RandString(ID_LEN)
	}

	return
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	_, err := ApplyPermissions(tx, initialConfig.Permissions)
	return err
}

func (u *User) AfterSave(tx *gorm.DB) (err error) {
	var destinations []Destination
	err = tx.Find(&destinations).Error
	if err != nil {
		return err
	}

	role := "view"
	if u.Admin {
		role = "cluster-admin"
	}

	for _, d := range destinations {
		var permission Permission
		err := tx.FirstOrCreate(&permission, &Permission{UserId: u.Id, DestinationId: d.Id, FromDefault: true}).Error
		if err != nil {
			return err
		}

		permission.Role = role

		err = tx.Save(&permission).Error
		if err != nil {
			return err
		}
	}

	return
}

// TODO (jmorganca): use foreign constraints instead?
func (u *User) BeforeDelete(tx *gorm.DB) error {
	err := tx.Model(u).Association("Sources").Clear()
	if err != nil {
		return err
	}
	err = tx.Where(&Token{UserId: u.Id}).Delete(&Token{}).Error
	if err != nil {
		return err
	}
	return tx.Where(&Permission{UserId: u.Id}).Delete(&Permission{}).Error
}

func (r *Destination) BeforeCreate(tx *gorm.DB) (err error) {
	if r.Id == "" {
		r.Id = generate.RandString(ID_LEN)
	}

	return
}

func (d *Destination) AfterCreate(tx *gorm.DB) error {
	_, err := ApplyPermissions(tx, initialConfig.Permissions)
	return err
}

func (d *Destination) AfterSave(tx *gorm.DB) (err error) {
	var users []User
	err = tx.Find(&users).Error
	if err != nil {
		return err
	}

	for _, u := range users {
		role := "view"
		if u.Admin {
			role = "cluster-admin"
		}

		var permission Permission
		err := tx.FirstOrCreate(&permission, &Permission{UserId: u.Id, DestinationId: d.Id, Role: role, FromDefault: true}).Error
		if err != nil {
			return err
		}
	}

	return
}

// TODO (jmorganca): use foreign constraints instead?
func (d *Destination) BeforeDelete(tx *gorm.DB) (err error) {
	return tx.Where(&Permission{DestinationId: d.Id}).Delete(&Permission{}).Error
}

func (g *Permission) BeforeCreate(tx *gorm.DB) (err error) {
	if g.Id == "" {
		g.Id = generate.RandString(ID_LEN)
	}

	return
}

func (s *Source) BeforeCreate(tx *gorm.DB) (err error) {
	if s.Id == "" {
		s.Id = generate.RandString(ID_LEN)
	}
	return
}

func (s *Source) BeforeDelete(tx *gorm.DB) error {
	var users []User
	if err := tx.Model(s).Association("Users").Find(&users); err != nil {
		return err
	}

	for _, u := range users {
		s.DeleteUser(tx, &u)
	}

	return nil
}

// CreateUser will create a user and associate them with the source
// If the user already exists, they will not be created, instead an association
// will be added instead
func (s *Source) CreateUser(db *gorm.DB, user *User, email string, password string, admin bool) error {
	var hashedPassword []byte
	var err error

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&User{Email: email}).Attrs(User{Admin: admin}).FirstOrCreate(&user).Error; err != nil {
			return err
		}

		if password != "" {
			hashedPassword, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return errors.New("could not create user")
			}

			user.Password = hashedPassword

			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		}

		if tx.Model(&user).Where(&Source{Id: s.Id}).Association("Sources").Count() == 0 {
			tx.Model(&user).Where(&Source{Id: s.Id}).Association("Sources").Append(s)
		}

		return nil
	})
}

// Delete will delete a user's association with a source
// If this is their only source, then the user will be deleted entirely
// TODO (jmorganca): wrap this in a transaction or at least find out why
// there seems to cause a bug when used in a nested transaction
func (s *Source) DeleteUser(db *gorm.DB, u *User) error {
	var user User

	if err := db.Where(&User{Email: u.Email}).First(&user).Error; err != nil {
		return err
	}

	if err := db.Model(&user).Association("Sources").Delete(s); err != nil {
		return err
	}

	if db.Model(&user).Association("Sources").Count() == 0 {
		if err := db.Delete(&user).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Source) SyncUsers(db *gorm.DB) error {
	var emails []string
	var err error

	switch s.Type {
	case "okta":
		emails, err = okta.Emails(s.OktaDomain, s.OktaClientId, s.OktaApiToken)
		if err != nil {
			return err
		}
	default:
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Create users in source
		for _, email := range emails {
			if err := s.CreateUser(tx, &User{}, email, "", false); err != nil {
				return err
			}
		}

		// Delete users not in source
		var toDelete []User
		if err := tx.Not("email IN ?", emails).Find(&toDelete).Error; err != nil {
			return err
		}

		for _, td := range toDelete {
			s.DeleteUser(tx, &td)
		}

		return nil
	})
}

func (s *Settings) BeforeCreate(tx *gorm.DB) (err error) {
	if s.Id == "" {
		s.Id = generate.RandString(ID_LEN)
	}
	return
}

func (s *Settings) BeforeSave(tx *gorm.DB) error {
	if len(s.PublicJWK) == 0 || len(s.PrivateJWK) == 0 {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}

		priv := jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.RS256), Use: "sig"}
		thumb, err := priv.Thumbprint(crypto.SHA256)
		if err != nil {
			return err
		}
		kid := base64.URLEncoding.EncodeToString(thumb)
		priv.KeyID = kid
		pub := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}

		privJS, err := priv.MarshalJSON()
		if err != nil {
			return err
		}

		pubJS, err := pub.MarshalJSON()
		if err != nil {
			return err
		}

		s.PrivateJWK = privJS
		s.PublicJWK = pubJS
	}
	return nil
}

func (t *Token) BeforeCreate(tx *gorm.DB) (err error) {
	if t.Id == "" {
		t.Id = generate.RandString(ID_LEN)
	}

	// TODO (jmorganca): 24 hours may be too long or too short for some teams
	// this should be customizable in settings or limited by the source's
	// policy (e.g. Okta is often 1-3 hours)
	if t.Expires == 0 {
		t.Expires = time.Now().Add(time.Hour * 24).Unix()
	}
	return
}

func (t *Token) CheckSecret(secret string) (err error) {
	h := sha256.New()
	h.Write([]byte(secret))
	if subtle.ConstantTimeCompare(t.Secret, h.Sum(nil)) != 1 {
		return errors.New("could not verify token secret")
	}

	return nil
}

func NewToken(db *gorm.DB, userId string, token *Token) (secret string, err error) {
	secret = generate.RandString(TOKEN_SECRET_LEN)

	h := sha256.New()
	h.Write([]byte(secret))
	token.UserId = userId
	token.Secret = h.Sum(nil)

	err = db.Create(token).Error
	if err != nil {
		return "", err
	}

	return
}

func (a *ApiKey) BeforeCreate(tx *gorm.DB) (err error) {
	if a.Id == "" {
		a.Id = generate.RandString(ID_LEN)
	}

	if a.Key == "" {
		a.Key = generate.RandString(API_KEY_LEN)
	}
	return
}

func NewDB(dbpath string) (*gorm.DB, error) {
	if err := os.MkdirAll(path.Dir(dbpath), os.ModePerm); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbpath), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Error,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})

	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&User{})
	db.AutoMigrate(&Source{})
	db.AutoMigrate(&Destination{})
	db.AutoMigrate(&Permission{})
	db.AutoMigrate(&Settings{})
	db.AutoMigrate(&Token{})
	db.AutoMigrate(&ApiKey{})

	// Add default source
	err = db.FirstOrCreate(&Source{}, &Source{Type: SOURCE_TYPE_INFRA}).Error
	if err != nil {
		return nil, err
	}

	// Add default settings
	err = db.FirstOrCreate(&Settings{}).Error
	if err != nil {
		return nil, err
	}

	return db, nil
}