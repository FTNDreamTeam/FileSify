package db

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/genproto/googleapis/type/date"
)

type FolderMeta struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Path    string             `bson:"name"`
	Created primitive.DateTime `bson:"created"`
	Owner   primitive.ObjectID `bson:"owner"`
}

type FileMeta struct {
	ID    primitive.ObjectID  `bson:"_id,omitempty"`
	Name  string              `bson:"name"`
	Date  date.Date           `bson:"date"`
	Time  primitive.Timestamp `bson:"time"`
	Owner primitive.ObjectID  `bson:"owner"`
	Path  string              `bson:"path"`
	Size  int32               `bson:"size"`
}
