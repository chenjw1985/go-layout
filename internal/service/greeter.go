package service

import (
	"context"

	v1 "github.com/davidchen-cn/go-layout/api/helloworld/v1"
	"github.com/davidchen-cn/go-layout/internal/biz"
	"github.com/pingcap/errors"
)

// GreeterService is a greeter service.
type GreeterService struct {
	v1.UnimplementedGreeterServer

	uc *biz.GreeterUsecase
}

// NewGreeterService new a greeter service.
func NewGreeterService(uc *biz.GreeterUsecase) *GreeterService {
	return &GreeterService{uc: uc}
}

// SayHello implements helloworld.GreeterServer.
func (s *GreeterService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	g, err := s.uc.CreateGreeter(ctx, &biz.Greeter{Hello: in.Name})
	if err != nil {
		return nil, errors.Wrap(err, "CreateGreeter")
	}
	return &v1.HelloReply{Message: "Hello " + g.Hello}, nil
}
