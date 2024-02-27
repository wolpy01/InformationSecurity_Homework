package storage

import (
	"context"
	"log"
	"proxyServer/mongo/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Storage struct {
	db            *mongo.Client
	reqCollection *mongo.Collection
}

func CreateStorage(db *mongo.Client) (Storage, error) {
	coll := db.Database("mitm").Collection("requests")
	_, err := coll.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys: bson.M{"host": 1},
	})
	if err != nil {
		log.Println("error Indexes CreateOne", err)
		return Storage{}, err
	}
	return Storage{db: db, reqCollection: coll}, nil
}

func (storage *Storage) Add(req domain.HTTPTransaction) error {
	_, err := storage.reqCollection.InsertOne(context.TODO(), req)
	if err != nil {
		log.Println("error insert one", err)
		return err
	}
	return nil
}

func (storage *Storage) GetAll() ([]domain.HTTPTransaction, error) {
	cur, err := storage.reqCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Println("error Find", err)
		return []domain.HTTPTransaction{}, err
	}
	defer cur.Close(context.TODO())

	var results []domain.HTTPTransaction
	for cur.Next(context.TODO()) {
		var result domain.HTTPTransaction
		err := cur.Decode(&result)
		if err != nil {
			log.Println("decode error", err)
			continue
		}
		id, ok := result.ID.(primitive.ObjectID)

		if ok {
			result.ID = id.Hex()
		} else {
			result.ID = "none"
		}

		results = append(results, result)
	}
	if err := cur.Err(); err != nil {
		log.Println("cur.Err error ", err)
		return []domain.HTTPTransaction{}, err
	}

	return results, nil
}

func (storage *Storage) GetByID(id string) (domain.HTTPTransaction, error) {

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.HTTPTransaction{}, err
	}

	transaction := domain.HTTPTransaction{}

	result := storage.reqCollection.FindOne(context.Background(), bson.M{"_id": objectId})
	err = result.Decode(&transaction)
	if err != nil {
		return domain.HTTPTransaction{}, err
	}

	return transaction, nil
}
