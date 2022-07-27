package client

import (
	"context"
	"reflect"
	"testing"
)

func TestPackModel(t *testing.T) {
	type args struct {
		ctx context.Context
		dir string
	}
	tests := []struct {
		name    string
		args    args
		want    *Model
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				ctx: context.Background(),
				dir: "testdata/test1",
			},
			want: &Model{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PackLocalModel(tt.args.ctx, tt.args.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("PackModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PackModel() = %v, want %v", got, tt.want)
			}
		})
	}
}
