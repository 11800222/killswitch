# mods




# build 
```shell
sudo make
```


```shell
go mod tidy

go mod vendor

make
```


# deploy
```shell
chmod +x killswitch

sudo chown root:wheel killswitch

sudo chmod 744 killswitch
```