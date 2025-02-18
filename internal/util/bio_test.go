package util

import (
	v1 "github.com/kyzrfranz/buntesdach/api/v1"
	"testing"
)

func TestLongSalutation(t *testing.T) {
	type args struct {
		bio v1.PoliticianBio
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "regular non gender",
			args: args{
				bio: v1.PoliticianBio{
					LastName:  "Moleman",
					FirstName: "Hans",
				},
			},
			want: "Hallo Hans Moleman",
		},
		{
			name: "regular male",
			args: args{
				bio: v1.PoliticianBio{
					LastName:  "Moleman",
					FirstName: "Hans",
					Gender:    "Männlich",
				},
			},
			want: "Sehr geehrter Herr Moleman",
		},
		{
			name: "regular female",
			args: args{
				bio: v1.PoliticianBio{
					LastName:  "Moleman",
					FirstName: "Hans",
					Gender:    "Weiblich",
				},
			},
			want: "Sehr geehrte Frau Moleman",
		},
		{
			name: "regular binary",
			args: args{
				bio: v1.PoliticianBio{
					LastName:  "Moleman",
					FirstName: "Hans",
					Gender:    "Binär",
				},
			},
			want: "Hallo Hans Moleman",
		},
		{
			name: "regular dr",
			args: args{
				bio: v1.PoliticianBio{
					LastName:      "Moleman",
					FirstName:     "Hans",
					Gender:        "Männlich",
					AcademicTitle: "Dr.",
				},
			},
			want: "Sehr geehrter Herr Dr. Moleman",
		},
		{
			name: "regular 'von'",
			args: args{
				bio: v1.PoliticianBio{
					LastName:      "Moleman",
					FirstName:     "Hans",
					Gender:        "Männlich",
					NobilityTitle: "von",
				},
			},
			want: "Sehr geehrter Herr von Moleman",
		},
		{
			name: "regular graf",
			args: args{
				bio: v1.PoliticianBio{
					LastName:      "Moleman",
					FirstName:     "Hans",
					Gender:        "Männlich",
					NobilityTitle: "Graf",
				},
			},
			want: "Sehr geehrter Herr Graf Moleman",
		},
		{
			name: "regular 'von'",
			args: args{
				bio: v1.PoliticianBio{
					LastName:      "Moleman",
					FirstName:     "Hans",
					Gender:        "Männlich",
					NobilityTitle: "Graf",
					AcademicTitle: "Prof.",
				},
			},
			want: "Sehr geehrter Herr Prof. Graf Moleman",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LongSalutation(tt.args.bio); got != tt.want {
				t.Errorf("LongSalutation() = %v, want %v", got, tt.want)
			}
		})
	}
}
