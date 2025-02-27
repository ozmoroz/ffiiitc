package main

import (
	"ffiiitc/internal/classifier"
	"ffiiitc/internal/config"
	"ffiiitc/internal/firefly"
	"ffiiitc/internal/handlers"
	"ffiiitc/internal/router"

	"github.com/go-pkgz/lgr"
)

const (
	ffAppTimeout = 10               // 10 sec for fftc to app service timeout
	modelFile    = "data/model.gob" //file name to store model
)

func main() {

	// make logger
	l := lgr.New(lgr.Debug, lgr.CallerFunc)
	l.Logf("INFO Firefly transaction classification started")

	// get the config
	l.Logf("INFO getting config")
	cfg, err := config.NewConfig()
	if err != nil {
		l.Logf("FATAL getting config: %v", err)
	}

	// make firefly http client for rest api
	fc := firefly.NewFireFlyHttpClient(cfg.FFApp, cfg.APIKey, ffAppTimeout, l)

	// make classifier
	// on first run, classifier will take all your
	// transactions and learn their categories
	// subsequent start classifier will load trained model from file
	l.Logf("INFO loading classifier from model: %s", modelFile)
	cls, err := classifier.NewTrnClassifierFromFile(modelFile, l)
	if err != nil {
		l.Logf("ERROR %v", err)
		l.Logf("INFO looks like we need to do some training...")
		// get transactions in data set
		//[ [cat, trn description], [cat, trn description]... ]
		trnDataset, err := fc.GetTransactionsDataset()
		if err != nil || len(trnDataset) == 0 {
			l.Logf("FATAL: %v", err)
		}
		l.Logf("DEBUG data set:\n %v", trnDataset)
		cls, err = classifier.NewTrnClassifierWithTraining(trnDataset, l)
		if err != nil {
			l.Logf("FATAL: %v", err)
		}
		l.Logf("INFO training completed...")
		err = cls.SaveClassifierToFile(modelFile)
		if err != nil {
			l.Logf("FATAL: %v", err)
		}
		l.Logf("INFO classifier saved to: %s", modelFile)
	}

	l.Logf("DEBUG learned classes: %v", cls.Classifier.Classes)

	// init handlers
	h := handlers.NewWebHookHandler(cls, fc, l)

	// init router
	r := router.NewRouter()

	// add handlers
	r.AddRoute("/classify", h.HandleNewTransactionWebHook)

	// temporary remove this handle
	//r.AddRoute("/learn", h.HandleUpdateTransactionWebHook)

	//run
	err = r.Run(8080)
	if err != nil {
		panic(err)
	}
}
