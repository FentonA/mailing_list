package grpcapi

import (
	"context"
	"database/sql"
	"log"
	"mailingList/mdb"
	pb "mailingList/proto"
	"net"
	"time"

	"google.golang.org/grpc"
)

type MailServer struct {
	pb.UnimplementedMailinglistServer
	db *sql.DB
}

func pbEntryToMdbEntry(pbEntry *pb.EmailEntry) mdb.EmailEntry {
	t := time.Unix(pbEntry.ConfirmedAt, 0)
	return mdb.EmailEntry{
		Id:        pbEntry.Id,
		Email:     pbEntry.Email,
		ConfirmAt: &t,
		OptOut:    pbEntry.OptOut,
	}
}

func mdbEntryToPbEntry(mdbEntyr *mdb.EmailEntry) pb.EmailEntry {
	return pb.EmailEntry{
		Id:          mdbEntyr.Id,
		Email:       mdbEntyr.Email,
		ConfirmedAt: mdbEntyr.ConfirmAt.Unix(),
		OptOut:      mdbEntyr.OptOut,
	}
}

func emailResponse(db *sql.DB, email string) (*pb.EmailResponse, error) {
	entry, err := mdb.GetEmail(db, email)
	if err != nil {
		return &pb.EmailResponse{}, err
	}
	if entry == nil {
		return &pb.EmailResponse{}, nil
	}

	res := mdbEntryToPbEntry(entry)

	return &pb.EmailResponse{EmailEntry: &res}, nil
}

func (s *MailServer) GetEmail(ctx context.Context, req *pb.GetEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("grp GetEmail: %v\n", req)
	return emailResponse(s.db, req.EmailAddr)
}

func (s *MailServer) GetEmailBatch(ctx context.Context, req *pb.GetEmailBatchRequest) (*pb.GetEmailRequestResponse, error) {
	log.Printf("gRpc GetEmailBatch: %v\n", req)

	params := mdb.GetEmailBatchQueryParams{
		Page:  int(req.Page),
		Count: int(req.Count),
	}
	mdbEntries, err := mdb.GetEmailBatch(s.db, params)
	if err != nil {
		return &pb.GetEmailRequestResponse{}, err
	}
	pbEntries := make([]*pb.EmailEntry, 0, len(mdbEntries))
	for i := 0; i < len(mdbEntries); i++ {
		entry := mdbEntryToPbEntry(&mdbEntries[i])
		pbEntries = append(pbEntries, &entry)
	}

	return &pb.GetEmailRequestResponse{EmailEntries: pbEntries}, nil
}

func (s *MailServer) CreateEmail(ctx context.Context, req *pb.CreateEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("gRPC CreateEmail: %v\n", req)

	err := mdb.CreateEmail(s.db, req.EmailAddr)
	if err != nil {
		return &pb.EmailResponse{}, err
	}
	return emailResponse(s.db, req.EmailAddr)
}

func (s *MailServer) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("gRPC CreateEmail: %v\n", req)

	entry := pbEntryToMdbEntry(req.EmailEntry)

	err := mdb.UpdateEmail(s.db, entry)
	if err != nil {
		return &pb.EmailResponse{}, err
	}

	return emailResponse(s.db, entry.Email)
}

func (s *MailServer) DeleteEmail(ctx context.Context, req *pb.DeleteEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("gRPC DeleteEmail : %v\n", req)

	err := mdb.DeleteEmail(s.db, req.EmailAddr)
	if err != nil {
		return &pb.EmailResponse{}, err
	}
	return emailResponse(s.db, req.EmailAddr)
}

func Serve(db *sql.DB, bind string) {
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("gRPC server error: failure to bind %v\n", bind)
	}

	grpcServer := grpc.NewServer()

	mailServer := MailServer{db: db}

	pb.RegisterMailinglistServer(grpcServer, &mailServer)

	log.Printf("gRPC API server listening on %v\n", bind)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("gRPC server error: %v\n", err)
	}
}
