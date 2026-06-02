<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Database {

    /**
     * @return array{host:string,port:?int,socket:?string,name:string,user:string,pass:string,charset:string}
     */
    public static function credentials() {
        return array(
            'host' => DB_HOST,
            'port' => null,
            'socket' => null,
            'name' => DB_NAME,
            'user' => DB_USER,
            'pass' => DB_PASSWORD,
            'charset' => defined('DB_CHARSET') && DB_CHARSET ? DB_CHARSET : 'utf8mb4',
        );
    }

    /**
     * @return array{host:string,port:?int,socket:?string}
     */
    public static function parse_host($host) {
        $port = null;
        $socket = null;
        $host_name = $host;

        if (false !== strpos($host, ':')) {
            $parts = explode(':', $host, 2);
            $host_name = $parts[0];
            $maybe = $parts[1];

            if (is_numeric($maybe)) {
                $port = (int) $maybe;
            } else {
                $socket = $maybe;
            }
        }

        return array(
            'host' => $host_name,
            'port' => $port,
            'socket' => $socket,
        );
    }

    public static function dsn() {
        $creds = self::credentials();
        $parsed = self::parse_host($creds['host']);

        if (!empty($parsed['socket'])) {
            return sprintf(
                'mysql:unix_socket=%s;dbname=%s;charset=%s',
                $parsed['socket'],
                $creds['name'],
                $creds['charset']
            );
        }

        if (!empty($parsed['port'])) {
            return sprintf(
                'mysql:host=%s;port=%d;dbname=%s;charset=%s',
                $parsed['host'],
                $parsed['port'],
                $creds['name'],
                $creds['charset']
            );
        }

        return sprintf(
            'mysql:host=%s;dbname=%s;charset=%s',
            $parsed['host'],
            $creds['name'],
            $creds['charset']
        );
    }

    public static function has_pdo_mysql() {
        return class_exists('PDO') && in_array('mysql', PDO::getAvailableDrivers(), true);
    }

    /**
     * @return wpdb
     */
    public static function wpdb() {
        global $wpdb;

        if (!isset($wpdb) || !($wpdb instanceof wpdb)) {
            throw new RuntimeException('WordPress database connection is unavailable.');
        }

        return $wpdb;
    }

    /**
     * @return string
     */
    public static function driver() {
        return self::has_pdo_mysql() ? 'pdo' : 'wpdb';
    }

    /**
     * @param bool $buffered
     * @return array<int,mixed>
     */
    public static function pdo_options($buffered = true) {
        if (!self::has_pdo_mysql()) {
            throw new RuntimeException('PDO MySQL is not available.');
        }

        return array(
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::MYSQL_ATTR_USE_BUFFERED_QUERY => $buffered,
        );
    }

    /**
     * @return PDO
     */
    public static function pdo() {
        if (!self::has_pdo_mysql()) {
            throw new RuntimeException('PDO MySQL is not available.');
        }

        $creds = self::credentials();
        return new PDO(
            self::dsn(),
            $creds['user'],
            $creds['pass'],
            self::pdo_options(true)
        );
    }

    /**
     * @param string $sql
     * @return int
     */
    public static function exec($sql) {
        if (self::has_pdo_mysql()) {
            return self::pdo()->exec($sql);
        }

        $wpdb = self::wpdb();
        $wpdb->query($sql);
        self::assert_no_db_error($wpdb);

        return (int) $wpdb->rows_affected;
    }

    /**
     * @param string $sql
     * @return array<int,array<string,mixed>>
     */
    public static function query_rows($sql) {
        if (self::has_pdo_mysql()) {
            $statement = self::pdo()->query($sql);
            if (false === $statement) {
                throw new RuntimeException('Query failed.');
            }

            return $statement->fetchAll(PDO::FETCH_ASSOC);
        }

        $wpdb = self::wpdb();
        $rows = $wpdb->get_results($sql, ARRAY_A);
        self::assert_no_db_error($wpdb);

        if (!is_array($rows)) {
            return array();
        }

        return $rows;
    }

    /**
     * @param wpdb $wpdb
     */
    public static function assert_no_db_error($wpdb) {
        if (!empty($wpdb->last_error)) {
            throw new RuntimeException($wpdb->last_error);
        }
    }

    public static function temp_dir() {
        $upload = wp_upload_dir();
        $base = isset($upload['basedir']) ? $upload['basedir'] : sys_get_temp_dir();
        $dir = trailingslashit($base) . 'dback-db-tools';

        if (!wp_mkdir_p($dir)) {
            throw new RuntimeException('Unable to create temporary directory.');
        }

        return $dir;
    }

    public static function temp_file($prefix, $suffix = '') {
        $dir = self::temp_dir();
        $path = $dir . '/' . $prefix . '-' . wp_generate_password(12, false, false) . $suffix;

        return $path;
    }
}
