package android

import (
	"testing"
)

const testData = `{"type":"notification","user_id":""}`

const testSeviceAccountJson = `{
	"type": "service_account",
	"project_id": "android-xxxxxy-app",
	"private_key_id": "9xxxxxxxxxxxxx",
	"private_key": "-----BEGIN PRIVATE KEY-----\n\n-----END PRIVATE KEY-----\n",
	"client_email": "firebase-adminsdk-fe5wa@android-libnotify-app.iam.gserviceaccount.com",
	"client_id": "11111111111111",
	"auth_uri": "https://accounts.google.com/o/oauth2/auth",
	"token_uri": "https://oauth2.googleapis.com/token",
	"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
	"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk-fe5wa%40android-libnotify-app.iam.gserviceaccount.com",
	"universe_domain": "googleapis.com"
}`

func TestPush(t *testing.T) {

	opts, err := MakeFcmMessageOpts(testSeviceAccountJson, 0)
	if err != nil {
		t.Error(err, "fcm: failed get jwt token")
	}

	sender := New()

	m := make(map[string]string)

	m["dry_run"] = "false"

	err = sender.Send("XXXX:XXXXXX", m, opts)

	if err != nil {
		t.Error(err)
	}

}
