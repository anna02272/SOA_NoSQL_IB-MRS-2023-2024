package data

import (
	"encoding/json"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Availability struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	AccommodationID  primitive.ObjectID `bson:"accommodation_id" json:"accommodation_id"`
	Date             time.Time          `bson:"date" json:"ddate"`
	Price            float64            `bson:"price" json:"price"`
	AvailabilityType AvailabilityType   `bson:"availability_type" json:"availability_type"`
}

type AvailabilityType string

const (
	Available   AvailabilityType = "Available"
	Unavailable AvailabilityType = "Unavailable"
	Booked      AvailabilityType = "Booked"
)

func (a *Availability) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(a)
}

func (a *Availability) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(a)
}