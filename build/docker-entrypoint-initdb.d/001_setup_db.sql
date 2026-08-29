/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `crons` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user` longtext DEFAULT NULL,
  `task` varchar(255) DEFAULT NULL,
  `cron_expression` varchar(100) DEFAULT NULL,
  `cron_name` longtext DEFAULT NULL,
  `server_id` bigint(20) DEFAULT NULL,
  `status` longtext DEFAULT NULL,
  `team_id` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_crons_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `failed_logins` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `date_time` datetime(3) DEFAULT NULL,
  `ip_address` varchar(100) DEFAULT NULL,
  `attempted_user_name` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_failed_logins_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `operation_logs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `entity_id` bigint(20) DEFAULT NULL,
  `entity` varchar(100) DEFAULT NULL,
  `log` longtext DEFAULT NULL,
  `log_level` varchar(100) DEFAULT NULL,
  `log_summary` varchar(255) DEFAULT NULL,
  `team_id` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_operation_logs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `scriptable_task_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `entity_id` bigint(20) DEFAULT NULL,
  `entity` longtext DEFAULT NULL,
  `task` longtext DEFAULT NULL,
  `task_status` varchar(20) DEFAULT 'complete',
  `team_id` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_scriptable_task_logs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;


CREATE TABLE `servers` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `server_name` varchar(100) DEFAULT NULL,
  `server_type` varchar(50) DEFAULT NULL,
  `server_ip` varchar(100) DEFAULT NULL,
  `private_server_ip` varchar(100) DEFAULT NULL,
  `ssh_username` varchar(100) DEFAULT NULL,
  `new_ssh_username` varchar(100) DEFAULT NULL,
  `ssh_port` bigint(20) DEFAULT NULL,
  `new_ssh_port` bigint(20) DEFAULT NULL,
  `redis` tinyint(3) DEFAULT NULL,
  `certbot` tinyint(3) DEFAULT NULL,
  `memcache` tinyint(3) DEFAULT NULL,
  `mysql` tinyint(3) DEFAULT NULL,
  `mysql_root_password` varchar(100) DEFAULT NULL,
  `php_version` varchar(100) DEFAULT NULL,
  `webserver_type` varchar(100) DEFAULT NULL,
  `scriptable_name` varchar(50) DEFAULT NULL,
  `status` varchar(100) DEFAULT NULL,
  `apt_packages` longtext DEFAULT NULL,
  `team_id` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_servers_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;


CREATE TABLE `sites` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `domain` varchar(255) DEFAULT NULL,
  `scriptable_name` varchar(100) DEFAULT NULL,
  `deploy_scriptables` varchar(100) DEFAULT NULL,
  `site_name` longtext DEFAULT NULL,
  `server_id` bigint(20) DEFAULT NULL,
  `webroot` varchar(100) DEFAULT NULL,
  `php_version` varchar(50) DEFAULT NULL,
  `lets_encrypt_certificate` tinyint(3) DEFAULT NULL,
  `mysql_password` varchar(155) DEFAULT NULL,
  `git_url` varchar(255) DEFAULT NULL,
  `branch` varchar(100) DEFAULT NULL,
  `environment` varchar(50) DEFAULT NULL,
  `status` varchar(50) DEFAULT NULL,
  `deploy_token` varchar(255) DEFAULT NULL,
  `team_id` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sites_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;


CREATE TABLE `teams` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(100) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;


CREATE TABLE `users` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `password` varchar(155) DEFAULT NULL,
  `verified` tinyint(3) DEFAULT NULL,
  `reset_token` varchar(155) DEFAULT NULL,
  `two_factor` tinyint(3) DEFAULT NULL,
  `two_factor_code` varchar(155) DEFAULT NULL,
  `team_id` int(11) DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

CREATE TABLE `site_queues` (
  `id` int NOT NULL AUTO_INCREMENT,
  `site_id` int DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `status` varchar(20) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `systemd_services` (
  `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
  `name` varchar(100) DEFAULT NULL,
  `site_id` int DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `status` varchar(20) DEFAULT NULL,
  `command` varchar(100) DEFAULT NULL,
  `scriptable` varchar(100) DEFAULT NULL,
  `team_id` int(11) DEFAULT 0
)  ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;