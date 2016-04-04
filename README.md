Chat is a go chat server.

### Features
* multiple rooms
* user logins
* multiple connection types
* optional database support
* block list
* friend list

### Config

There is a sample Config file provided.  The server will look for a config file in its folder. A different location can be specified using the -config _filename_ flag.  The server will start the connection types that have ports specified for them in the config.  Origin is the origin of the site serving the web interface to allow the CORS to work propery.

### Commands
/tell _user_ _message_ - send the message to the specified user  
/block _user_ - adds the user to your block list preventing future messages from that user  
/unblock _user_ - removes the user from your block list allowing messages from that user  
/friend _user_ - adds the user to your friend list  
/unfriend _user_ - removes the user from your friend list  
/friendlist - shows your friend list and displays what room your friends are in or when they last logged in  
/blocklist - shows you block list  
/join _room name_ - moves you to the specifed room or creates it if it doesn't exist *won't create the room if the room limit has been reached  
/quit - logges you out of the server  
/list - shows a list of the current rooms  

### Database

The server currently supports only a Postgresql database.  If no database is specified the user information will instead be stored in a file.  New database types can be added by creating an adapter that meets the DataStore interface in clientdata.go and then adding an entry in the datafactory package.

Browser http [client](https://github.com/DavidAFox/ChatWebInterface)

[Angularjs version](https://github.com/DavidAFox/WebChatInterfaceAJS)