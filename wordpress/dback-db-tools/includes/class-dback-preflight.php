<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Preflight {

    /**
     * @return array<string,mixed>
     */
    public static function run() {
        $checks = array();
        $success = true;

        $checks[] = self::check('php_version', version_compare(PHP_VERSION, '7.4', '>='), 'PHP ' . PHP_VERSION, 'PHP 7.4 or newer is required.');
        $checks[] = self::check('zlib', function_exists('gzopen') || function_exists('gzencode'), 'gzip available', 'zlib/gzip support is required.');
        $checks[] = self::check('pdo_mysql', DBack_Database::has_pdo_mysql() || class_exists('wpdb'), DBack_Database::driver(), 'PDO MySQL or wpdb fallback is required.');
        $checks[] = self::check('uploads_writable', self::uploads_writable(), 'uploads writable', 'WordPress uploads directory must be writable.');
        $checks[] = self::check('temp_dir', self::temp_dir_writable(), 'temp dir ready', 'Unable to create plugin temp directory under uploads.');

        $db_version = '';
        try {
            $rows = DBack_Database::query_rows('SELECT 1 AS ok');
            $db_ok = !empty($rows);
            if ($db_ok && isset($rows[0]['ok'])) {
                $db_version = 'SELECT 1 ok';
            }
            $checks[] = self::check('database', $db_ok, $db_version !== '' ? $db_version : DBack_Database::driver(), 'Database query failed.');
        } catch (Throwable $exception) {
            $checks[] = self::check('database', false, $exception->getMessage(), 'Database query failed.');
        }

        foreach ($checks as $check) {
            if ('fail' === $check['status']) {
                $success = false;
            }
        }

        return array(
            'success' => $success,
            'plugin_version' => DBACK_DB_TOOLS_VERSION,
            'php_version' => PHP_VERSION,
            'wordpress_version' => get_bloginfo('version'),
            'site_url' => site_url(),
            'driver' => DBack_Database::driver(),
            'db_version' => $db_version,
            'checks' => $checks,
        );
    }

    private static function check($name, $ok, $details_ok, $details_fail) {
        return array(
            'name' => $name,
            'status' => $ok ? 'ok' : 'fail',
            'details' => $ok ? $details_ok : $details_fail,
        );
    }

    private static function uploads_writable() {
        $upload = wp_upload_dir();
        if (empty($upload['basedir'])) {
            return false;
        }

        return is_writable($upload['basedir']);
    }

    private static function temp_dir_writable() {
        try {
            DBack_Database::temp_dir();
            return true;
        } catch (Throwable $exception) {
            return false;
        }
    }
}
