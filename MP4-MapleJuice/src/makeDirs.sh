mv shared/inputData inputData
rm -r -f shared/*
mv inputData shared/inputData
mkdir -p shared/inputData
mkdir -p shared/outputData
mkdir -p shared/SDFS
mkdir -p shared/shardedMapInputData
mkdir -p shared/shardedMapOutputData
mkdir -p shared/shardedReduceInputData
mkdir -p shared/shardedReduceOutputData
