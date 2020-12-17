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

## Usage
REST API has only one endpoint:

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

**fileUrl** and **language** is required.
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
