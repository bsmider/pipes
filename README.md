# pipes

### Running w/ docker

in root directory:
1. make generate-protos
2. docker-compose up --build core-example

this will spawn a docker container to do code & dockerfile generation. the generated artifacts can be found in the container at /app/example/generated if you want to inspect.

once the generation is done, another container will be spawned to compile the generated code into binaries, and then it will run the orchestrator binary.