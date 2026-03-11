create extension if not exists "pgcrypto";

create table users (
	id uuid primary key default gen_random_uuid(),
	name varchar(255) not null,
	email varchar(255) unique not null,
	password_hash text not null,
	created_at timestamp default now()
);

create table products (
	id uuid primary key default gen_random_uuid(),
	name varchar(255) not null,
	description text,
	price numeric(12,2) not null,
	created_at timestamp default now()
);

create table product_inventory (
	product_id uuid primary key,
	stock int not null check (stock >= 0),
	reserved int default 0 check (reserved >= 0),
	updated_at timestamp default now(),
	foreign key (product_id) references products(id) on delete cascade
);

create table orders (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null,
	status varchar(50) not null,
	total_amount numeric (12, 2),
	created_at timestamp default now(),
	foreign key (user_id) references users(id) 
);

create table order_items (
	id uuid primary key default gen_random_uuid(),
	order_id uuid not null,
	product_id uuid not null,
	quantity int not null,
	price numeric(12,2) not null,
	foreign key (order_id) references orders(id),
	foreign key (product_id) references products(id)
);

create table payments (
	id uuid primary key default gen_random_uuid(),
	order_id uuid not null,
	status varchar(50) not null,
	payment_method varchar(50),
	paid_at timestamp ,
	foreign key (order_id) references orders(id)
);
