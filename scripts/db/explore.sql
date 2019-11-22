DROP DATABASE IF EXISTS `explore`;
CREATE DATABASE `explore` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
GRANT ALL PRIVILEGES ON explore.* TO 'root'@'localhost';
FLUSH PRIVILEGES;
USE `explore`;

CREATE TABLE IF NOT EXISTS `transaction` (
`tx_index` BIGINT UNSIGNED  NOT NULL   COMMENT '',
`block_number` BIGINT UNSIGNED NOT NULL   COMMENT '',
`tx_id` VARCHAR(64)  NOT NULL   COMMENT '',
`peer` VARCHAR(64)  NOT NULL   COMMENT '',
`tx_type` INT  NOT NULL   COMMENT '',
`status` INT  NOT NULL   COMMENT '',
`datetime` TIMESTAMP  NOT NULL   COMMENT '',
PRIMARY KEY (`tx_index`)
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `peer` (
`name` VARCHAR(50)  NOT NULL   COMMENT '',
`peer_type` INT  NOT NULL   COMMENT '',
`peer_status` INT  NOT NULL   COMMENT '',
PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `block` (
`number` BIGINT UNSIGNED NOT NULL   COMMENT '',
`block_hash` VARCHAR(64)  NOT NULL   COMMENT '',
`tx_count` INT  NOT NULL   COMMENT '',
`datetime` TIMESTAMP  NOT NULL   COMMENT '',
`alice_balance` VARCHAR(64)  NOT NULL   COMMENT '',
`bob_balance` VARCHAR(64)  NOT NULL   COMMENT '',
`block_type` INT  NOT NULL   COMMENT '',
PRIMARY KEY (`number`)
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
