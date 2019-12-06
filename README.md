# fabric-byzantine

# Start Fabric Network
```
docker-compose up -d
```

# Initialize Fabric Network
```
 docker exec -it cli bash

./scripts/script.sh
```
Note: 时间较长，耐心等待，等初始化完成后再启动其他服务

# Run Mysql server
``` 
docker-compose -f docker-compose-mysql.yaml up -d
```

# Run Byzantine server
```
docker-compose -f docker-compose-server.yaml up -d
```

