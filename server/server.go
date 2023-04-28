package server

import (
	"database/sql"
	"fmt"
	handlerusersparent "github.com/RyaWcksn/nann-e/api/v1/handler/authentication"
	serviceusersparent "github.com/RyaWcksn/nann-e/api/v1/service/authentication"
	"github.com/RyaWcksn/nann-e/pkgs/database/mysql"
	"github.com/RyaWcksn/nann-e/server/middleware"
	storeusersparent "github.com/RyaWcksn/nann-e/store/database/user"
	"github.com/gofiber/fiber/v2"
	"os"
	"strconv"

	"github.com/RyaWcksn/nann-e/config"
	"github.com/RyaWcksn/nann-e/pkgs/logger"
)

type Server struct {
	cfg *config.Config
	log logger.ILogger

	// Users Parent
	serviceUsersParent serviceusersparent.IService
	handlerUsersParent handlerusersparent.IHandler
}

var addr string
var SVR *Server
var db *sql.DB
var signalChan chan (os.Signal) = make(chan os.Signal, 1)
var ViberApp *fiber.App

func (s *Server) initServer() {
	addr = ":9000"
	cfg := s.cfg
	if len(cfg.Server.HTTPAddress) > 0 {
		if _, err := strconv.Atoi(cfg.Server.HTTPAddress); err == nil {
			addr = fmt.Sprintf(":%v", cfg.Server.HTTPAddress)
		} else {
			addr = cfg.Server.HTTPAddress
		}
	}
}

func (s *Server) Register() {
	s.initServer()

	// MYSQL
	dbConn := mysql.NewDatabaseConnection(*s.cfg, s.log)
	if dbConn == nil {
		s.log.Fatal("Expecting DB connection but received nil")
	}

	db = dbConn.DBConnect()
	if db == nil {
		s.log.Fatal("Expecting DB connection but received nil")
	}

	usersParentRepo := storeusersparent.NewUserParentImpl(db, s.log)

	// Register service
	s.serviceUsersParent = serviceusersparent.NewServiceImpl(usersParentRepo, s.cfg, s.log)

	// Register handler
	s.handlerUsersParent = handlerusersparent.NewUsersParentHandler(s.serviceUsersParent, s.log)
}

func New(cfg *config.Config, logger logger.ILogger) *Server {
	if SVR != nil {
		return SVR
	}
	SVR = &Server{
		cfg: cfg,
		log: logger,
	}

	SVR.Register()

	return SVR
}

func (s Server) Start() {
	ViberApp = fiber.New(fiber.Config{
		Immutable: true,
	})

	v1 := ViberApp.Group("/api/v1")
	v1.Use(middleware.ErrorHandler)
	v1.Post("/user/register", s.handlerUsersParent.RegisterParent)
	v1.Post("/user/login", s.handlerUsersParent.LoginParent)

	go func() {
		err := ViberApp.Listen(":9000")
		if err != nil {
			s.log.Fatalf("error listening to address %v, err=%v", addr, err)
		}
		s.log.Infof("HTTP server started %v", addr)
	}()

	sig := <-signalChan
	s.log.Infof("%s signal caught", sig)

	// Doing cleanup if received signal from Operating System.
	err := db.Close()
	if err != nil {
		s.log.Errorf("Error in closing DB connection. Err : %+v", err.Error())
	}
}
