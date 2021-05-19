# Speech to text Service
This service implement **REST API** for recognition audio files using google cloud speech recognition

## Installation

1. First you need to have [Google Cloud account](https://cloud.google.com/ "Google Cloud")
2. Create credentials.json file according to [documentation](https://cloud.google.com/docs/authentication/getting-started)
3. Enable [speech-to-text](https://cloud.google.com/speech-to-text) service (press "Go to console" button and enable Cloud Speech-to-Text API)
4. Create google bucket using [documentation](https://cloud.google.com/storage/docs/creating-buckets)
5. Run `export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json`, where `/path/to/credentials.json` is path 
   to json file from p.2
6. Run `export BUCKET_NAME=you_bucket`, where `you_bucket` is bucket name from 
   to json file from p.4
7. Run `docker-compose build`
8. Run `docker-compose up -d`. For watching the application logs use `docker-compose logs -f` command

## Google credentials
To run the application, a prerequisite is the presence of two environment variables. 
`GOOGLE_APPLICATION_CREDENTIALS` - The path to the main credentials file, which will be used for all enterpises, for 
which they are not specified in a separate config (see below about it).<br>
`BUCKET_NAME` - Also a required variable and the name of the bucket, which will be used for the name of the bucket, 
in case it is not configured separately for enterprises.
### Add credentials for specific enterprises
there are 2 ways to add credentials. 
1. Using config
2. Using API (which adds these configs to the config in p.1)

#### Using config
`config/buckets.json` should contain the configuration for buckets, for example:
```
{
  "3": "bucket-name-for-enterprise-3",
  "4": "bucket-name-for-enterprise-4"
}
```
`config/credentials/<enterprise_id>.json` should contain credentials files
for example `config/credentials/3.json` - is credentials for enterprise with id `3`
#### Using API
POST `localhost:7070/getTexts`
___
### Body Example:
```
{
   "credentials":{
      "auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs",
      "auth_uri":"https://accounts.google.com/o/oauth2/auth",
      "client_email":"speech-to-text@seraphic-vertex-234234.iam.gserviceaccount.com",
      "client_id":"12341234123412341234",
      "client_x509_cert_url":"https://www.googleapis.com/robot/v1/metadata/x509/speech-to-text%40seraphic-vertex-234234.iam.gserviceaccount.com",
      "private_key":"-----BEGIN PRIVATE KEY-----\nSomePrivateKey...\n-----END PRIVATE KEY-----\n",
      "private_key_id":"SomePrivateKey",
      "project_id":"seraphic-vertex-234234",
      "token_uri":"https://oauth2.googleapis.com/token",
      "type":"service_account"
      }, 
   "bucketName":"someBucketName", 
   "enterpriseId": 1
}
```
This endpoint checks for the presence of a bucket and the validity of the credentials. 
If something is invalid, the server will return a 409 error code. In case successful addition, 200 code is returned.
<hr>

## Speech to text Usage
REST API has only one endpoint for speech recognition:

POST `localhost:7070/getTexts`
___
### Body Example:
```
[
  {
    "uuid": "A23D3",
    "fileUrl": "https://some-site.com/some_audio.wav",
    "language": "en"
  },
  {
    "fileUrl": "https://some-site.com/some_audio2.wav",
    "language": "en"
  }
]
```
**uuid** is optional.

**fileUrl** and **language** is required. Supported languages: ***en, it, de, fr, nl, es, ca, gl, pt, pl, ro, el, da, 
eu, ru, bg, sl, sr, hr.***

___
### Response Example:
```
[
  {
    "uuid": "A23D3",
    "fileUrl": "https://some-site.com/some_audio.wav",
    "text": "Good morning, and welcome to WWDC. WDC is incredibly important and our users..",
    "duration": 15,
    "language": "en"
  },
  {
    "uuid": "",
    "fileUrl": "https://some-site.com/some_audio2.wav",
    "text": "It's sure that we bring some of our biggest. I have a chance to live and we have not stopped, <.....>",
    "duration": 45,
    "language": "en"
  }
]
```
**text** is recogtized text

**duration** is duration of vaw file which is a multiple of **15**, according to Google tariffication
