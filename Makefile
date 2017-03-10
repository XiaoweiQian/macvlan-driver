PLUGIN_NAME=macvlan_swarm

all: clean compile rootfs create

clean:
	@echo "### rm ./plugin"
	@rm -rf ./plugin

compile:
	@echo "### compile docker-macvlan plugin"
	@docker build -q -t builder -f Dockerfile.dev .
	@echo "### extract docker-macvlan"
	@docker create --name tmp builder
	@docker cp tmp:/go/bin/macvlan-driver ./docker-macvlan
	@docker rm -vf tmp
	@docker rmi builder

rootfs:
	@echo "### docker build: rootfs image with docker-macvlan"
	@docker build -q -t ${PLUGIN_NAME}:rootfs .
	@echo "### create rootfs directory in ./plugin/rootfs"
	@mkdir -p ./plugin/rootfs
	@docker create --name tmp ${PLUGIN_NAME}:rootfs
	@docker export tmp | tar -x -C ./plugin/rootfs
	@echo "### copy config.json to ./plugin/"
	@cp config.json ./plugin/
	@docker rm -vf tmp
	@docker rmi ${PLUGIN_NAME}:rootfs 

create:
	@echo "### remove existing plugin ${PLUGIN_NAME} if exists"
	@docker plugin rm -f ${PLUGIN_NAME} || true
	@echo "### create new plugin ${PLUGIN_NAME} from ./plugin"
	@docker plugin create ${PLUGIN_NAME} ./plugin

enable:
	@echo "### enable plugin ${PLUGIN_NAME}"
	@docker plugin enable ${PLUGIN_NAME}

push:  clean compile rootfs create enable
	@echo "### push plugin ${PLUGIN_NAME}"
	@docker plugin push ${PLUGIN_NAME}
