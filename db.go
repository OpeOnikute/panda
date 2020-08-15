package panda

import (
	"context"
	"time"

	"github.com/opeonikute/panda/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gopkg.in/mgo.v2/bson"
)

type db struct {
	Database    *mongo.Database
	Collections *dBCollection
}

type dBCollection struct {
	Entries *mongo.Collection
}

// Entry ...
type Entry struct {
	ID           primitive.ObjectID `json:"_id" bson:"_id"`
	URL          string             `json:"url" bson:"url"`
	Source       string             `json:"source" bson:"source"`
	FileName     string             `json:"filename" bson:"filename"`
	WordOfTheDay string             `json:"wordOfTheDay" bson:"wordOfTheDay"`
	Date         time.Time          `json:"date" bson:"date"`
	Created      time.Time          `json:"created" bson:"created"`
	Updated      time.Time          `json:"updated" bson:"updated"`
}

// Connect creates a new DB connection
func (db *db) Connect(mongoURL string, mongoDB string) error {

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURL))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(context.TODO())
	if err != nil {
		return err
	}

	db.Database = client.Database(mongoDB)

	db.Collections = new(dBCollection)
	db.Collections.Entries = db.Database.Collection("entries")

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// confirm we can connect to the db
	err = client.Ping(ctx, readpref.Primary())

	if err != nil {
		return err
	}

	return nil
}

// POD - Panda Of The Day
func (db *db) InsertPOD(newEn Entry) (Entry, error) {

	// TODO: Ensure we have already connected to the database here

	// if POD already exists, update the image
	date := getTodaysDate()
	existingEn, err := db.FindPOD(date)

	if err != nil {
		return newEn, err
	}

	// confirm that the entry truly doesn't exist
	if existingEn.URL != "" && existingEn.Source != "" {
		existingEn.URL = newEn.URL
		existingEn.Source = newEn.Source
		existingEn.FileName = newEn.FileName
		existingEn.WordOfTheDay = util.GetDailyWord()
		return db.updatePOD(existingEn)
	}

	// insert new entry with POD
	newEn.ID = primitive.NewObjectID()
	newEn.WordOfTheDay = util.GetDailyWord()
	newEn.Date = getTodaysDate()
	newEn.Created = time.Now()
	newEn.Updated = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.Collections.Entries.InsertOne(ctx, newEn)
	return newEn, err
}

// Uses today's date to find the POD
func (db *db) FindPOD(date time.Time) (Entry, error) {
	var en Entry
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.Collections.Entries.FindOne(ctx, bson.M{"date": date}).Decode(&en)

	if err != nil {
		if err.Error() == mongo.ErrNoDocuments.Error() {
			return en, nil
		}
		return en, err
	}

	return en, err
}

// updates image or whatever else we need
// works by passing in the updated model
func (db *db) updatePOD(en Entry) (Entry, error) {
	en.Updated = time.Now()

	update := make(map[string]interface{})
	update["$set"] = en

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Collections.Entries.UpdateOne(ctx, bson.M{"_id": en.ID}, update)
	return en, err
}

// creates a time instance and sets it to 12am to get the date
func getTodaysDate() time.Time {
	// MM-DD-YYYY
	tm := time.Now()
	return GetDate(tm)
}

// GetDate converts a time to 12am so it becomes a date
func GetDate(tm time.Time) time.Time {
	return time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, tm.Location())
}
