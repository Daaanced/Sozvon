CREATE TABLE chats 
( id UUID PRIMARY KEY, 
active BOOLEAN NOT NULL DEFAULT FALSE, 
created_at TIMESTAMP NOT NULL DEFAULT NOW() ); 

CREATE TABLE chat_members 
( chat_id UUID REFERENCES chats(id) ON DELETE CASCADE, 
login TEXT NOT NULL, PRIMARY KEY (chat_id, login) ); 

CREATE TABLE messages 
( id UUID PRIMARY KEY, 
chat_id UUID REFERENCES chats(id) ON DELETE CASCADE, 
sender_login TEXT NOT NULL, text TEXT NOT NULL, 
created_at TIMESTAMP NOT NULL DEFAULT NOW() ); 

CREATE INDEX idx_messages_chat_id_created ON messages(chat_id, created_at);