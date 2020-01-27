## How to Run

To run MP3 first "export GOPATH=/home/USERNAME/WORKSPACE/MP3"
To run MP4 first "export GOPATH=/home/USERNAME/WORKSPACE/MP4"

### Word Count
* start worker with "go run worker.go wordCount 1&"
* start map with "go run masterMap.go wordCount shared/inputData/wordcountbig.txt shared/shardedMapInputData/ shared/shardedMapOutputData/" for large dataset
* start map with "go run masterMap.go wordCount shared/inputData/wordCountInput.txt shared/shardedMapInputData/ shared/shardedMapOutputData/" for small dataset
* start reducer with "go run masterReducer.go wordCount 2 shared/shardedMapOutputData/ shared/shardedReduceInputData/ shared/shardedReduceOutputData/ shared/outputData/output.txt"

### Reverse Links
* start worker with "go run worker.go reverse 1&"
* start map with "go run masterMap.go reverse shared/inputData/reverseFullDataset.txt shared/shardedMapInputData/ shared/shardedMapOutputData/" for large dataset
* start map with "go run masterMap.go reverse shared/inputData/reverseInput.txt shared/shardedMapInputData/ shared/shardedMapOutputData/"
* start reducer with "go run masterReducer.go reverse 2 shared/shardedMapOutputData/ shared/shardedReduceInputData/ shared/shardedReduceOutputData/ shared/outputData/output.txt"