<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Database {

    /** @var string|null */
    private static $active_database = null;

    /**
     * @return array{host:string,port:?int,socket:?string,name:string,user:string,pass:string,charset:string}
     */
    public static function credentials() {
        return array(
            'host' => DB_HOST,
            'port' => null,
            'socket' => null,
            'name' => self::active_database_name(),
            'user' => DB_USER,
            'pass' => DB_PASSWORD,
            'charset' => defined('DB_CHARSET') && DB_CHARSET ? DB_CHARSET : 'utf8mb4',
        );
    }

    /**
     * @return string
     */
    public static function active_database_name() {
        if (is_string(self::$active_database) && '' !== self::$active_database) {
            return self::$active_database;
        }

        return DB_NAME;
    }

    /**
     * @param mixed $requested
     * @return string
     */
    public static function resolve_import_database($requested) {
        $requested = trim((string) $requested);
        if ('' === $requested) {
            return DB_NAME;
        }

        self::validate_database_name($requested);

        return $requested;
    }

    /**
     * @param string $name
     */
    public static function validate_database_name($name) {
        if (!preg_match('/^[A-Za-z0-9_]{1,64}$/', $name)) {
            throw new InvalidArgumentException('Invalid database name.');
        }
    }

    /**
     * @param string $name
     * @return string
     */
    public static function quote_identifier($name) {
        return '`' . str_replace('`', '``', $name) . '`';
    }

    /**
     * Select the database used for a restore/import operation.
     *
     * @param mixed $requested Empty string means the WordPress default database.
     * @return string
     */
    public static function prepare_import_target($requested) {
        $target = self::resolve_import_database($requested);
        if ($target !== DB_NAME) {
            self::create_database_if_missing($target);
        }
        self::activate_database($target);

        return $target;
    }

    /**
     * Run a callback while connected to the requested import/query database.
     *
     * @param mixed $requested
     * @param callable(string):mixed $callback
     * @return mixed
     */
    public static function with_target_database($requested, $callback) {
        $target = self::prepare_import_target($requested);

        try {
            return $callback($target);
        } finally {
            self::reset_import_target();
        }
    }

    public static function reset_import_target() {
        self::$active_database = null;
        if (self::has_pdo_mysql()) {
            return;
        }

        $wpdb = self::wpdb();
        $wpdb->select(DB_NAME);
        if ($wpdb->ready) {
            $wpdb->dbname = DB_NAME;
        }
    }

    /**
     * @param string $name
     */
    private static function activate_database($name) {
        self::$active_database = $name;

        if (self::has_pdo_mysql()) {
            return;
        }

        $wpdb = self::wpdb();
        $wpdb->select($name);

        if (!$wpdb->ready) {
            throw new RuntimeException(self::select_database_error($name, $wpdb));
        }

        $current = (string) $wpdb->get_var('SELECT DATABASE()');
        if ($current !== $name) {
            throw new RuntimeException(
                sprintf(
                    'Unable to select database "%s". Active connection is using "%s".',
                    $name,
                    $current
                )
            );
        }

        $wpdb->dbname = $name;
    }

    /**
     * @param string $name
     * @param wpdb $wpdb
     * @return string
     */
    private static function select_database_error($name, $wpdb) {
        $detail = trim((string) $wpdb->last_error);
        if ('' !== $detail) {
            return sprintf('Unable to select database "%s": %s', $name, $detail);
        }

        return sprintf(
            'Unable to select database "%s". The WordPress database user may lack privileges on this database.',
            $name
        );
    }

    /**
     * @param string $name
     */
    private static function create_database_if_missing($name) {
        if (self::database_exists($name)) {
            return;
        }

        $quoted = self::quote_identifier($name);
        $saved = self::$active_database;
        self::$active_database = null;

        try {
            if (self::has_pdo_mysql()) {
                self::pdo()->exec('CREATE DATABASE IF NOT EXISTS ' . $quoted);
                return;
            }

            $wpdb = self::wpdb();
            $wpdb->query('CREATE DATABASE IF NOT EXISTS ' . $quoted);
            self::assert_no_db_error($wpdb);
        } finally {
            self::$active_database = $saved;
        }
    }

    /**
     * @param string $name
     * @return bool
     */
    private static function database_exists($name) {
        $saved = self::$active_database;
        self::$active_database = null;

        try {
            if (self::has_pdo_mysql()) {
                $statement = self::pdo()->query('SHOW DATABASES LIKE ' . self::quote_sql_string($name));
                if (false === $statement) {
                    return false;
                }

                return (bool) $statement->fetch();
            }

            $wpdb = self::wpdb();
            $found = $wpdb->get_var('SHOW DATABASES LIKE ' . self::quote_sql_string($name));
            self::assert_no_db_error($wpdb);

            return is_string($found) && $found === $name;
        } catch (Throwable $exception) {
            return false;
        } finally {
            self::$active_database = $saved;
        }
    }

    /**
     * @param string $value
     * @return string
     */
    private static function quote_sql_string($value) {
        return "'" . str_replace("'", "''", $value) . "'";
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
