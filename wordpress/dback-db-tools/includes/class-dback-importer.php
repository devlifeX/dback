<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Importer {

    /**
     * Import SQL from a gzip-compressed file path.
     *
     * @param string $path
     * @param string $target_database Optional import target; empty uses the WordPress default database.
     * @return array{success:bool,statements_executed:int,bytes_received:int,database:string}
     * @throws Exception
     */
    public static function import_gzip_file($path, $target_database = '') {
        if (!is_readable($path)) {
            throw new RuntimeException('Import file is not readable.');
        }

        if (!function_exists('gzopen')) {
            throw new RuntimeException('The zlib extension is required for gzip import.');
        }

        @set_time_limit(0);
        if (function_exists('ini_set')) {
            @ini_set('memory_limit', '512M');
        }

        $selected_database = DBack_Database::prepare_import_target($target_database);
        $verify_import = self::is_verify_database($selected_database);

        try {
            $handle = gzopen($path, 'rb');
            if (false === $handle) {
                throw new RuntimeException('Unable to open gzip import file.');
            }

            try {
                $executed = self::execute_stream($handle, $verify_import);
            } finally {
                gzclose($handle);
            }

            if (0 === $executed) {
                throw new RuntimeException('No SQL statements were executed from the import file.');
            }

            return array(
                'success' => true,
                'statements_executed' => $executed,
                'bytes_received' => filesize($path),
                'database' => $selected_database,
            );
        } finally {
            DBack_Database::reset_import_target();
        }
    }

    /**
     * Import SQL from a WordPress REST request (preferred).
     *
     * WP_REST_Server reads php://input before the route callback runs, so the
     * gzip payload must be taken from WP_REST_Request::get_body().
     *
     * @param WP_REST_Request $request
     * @return array{success:bool,statements_executed:int,bytes_received:int}
     * @throws Exception
     */
    public static function import_rest_request($request) {
        $body = '';
        $database = '';
        if ($request instanceof WP_REST_Request) {
            $body = $request->get_body();
            $database = (string) $request->get_header('X-DBACK-DATABASE');
            if ('' === $database) {
                $database = (string) $request->get_param('database');
            }
        }

        if ('' === $body) {
            return self::import_request_body($database);
        }

        return self::import_gzip_payload($body, $database);
    }

    /**
     * Import SQL from the raw request body (gzip stream).
     *
     * @param string $target_database
     * @return array{success:bool,statements_executed:int,bytes_received:int,database:string}
     * @throws Exception
     */
    public static function import_request_body($target_database = '') {
        @set_time_limit(0);
        if (function_exists('ini_set')) {
            @ini_set('memory_limit', '512M');
        }

        $temp_file = DBack_Database::temp_file('import', '.sql.gz');
        $input = fopen('php://input', 'rb');
        $output = fopen($temp_file, 'wb');

        if (false === $input || false === $output) {
            throw new RuntimeException('Unable to read import payload.');
        }

        try {
            stream_copy_to_stream($input, $output);
        } finally {
            fclose($input);
            fclose($output);
        }

        try {
            return self::import_gzip_file($temp_file, $target_database);
        } finally {
            if (file_exists($temp_file)) {
                unlink($temp_file);
            }
        }
    }

    /**
     * @param string $payload
     * @param string $target_database
     * @return array{success:bool,statements_executed:int,bytes_received:int,database:string}
     * @throws Exception
     */
    private static function import_gzip_payload($payload, $target_database = '') {
        @set_time_limit(0);
        if (function_exists('ini_set')) {
            @ini_set('memory_limit', '512M');
        }

        $bytes = strlen($payload);
        if ($bytes < 3) {
            throw new RuntimeException('Import payload is empty.');
        }
        if ("\x1f\x8b" !== substr($payload, 0, 2)) {
            throw new RuntimeException('Import payload is not a valid gzip stream.');
        }

        $temp_file = DBack_Database::temp_file('import', '.sql.gz');
        if (false === file_put_contents($temp_file, $payload, LOCK_EX)) {
            throw new RuntimeException('Unable to save import payload.');
        }

        try {
            $result = self::import_gzip_file($temp_file, $target_database);
            $result['bytes_received'] = $bytes;
            return $result;
        } finally {
            if (file_exists($temp_file)) {
                unlink($temp_file);
            }
        }
    }

    /**
     * @param resource $handle
     * @return int
     */
    private static function execute_stream($handle, $verify_import = false) {
        $buffer = '';
        $executed = 0;

        while (!feof($handle)) {
            $line = fgets($handle);
            if (false === $line) {
                break;
            }

            $trimmed = trim($line);
            if ('' === $trimmed || 0 === strpos($trimmed, '--') || 0 === strpos($trimmed, '#')) {
                continue;
            }

            $buffer .= $line;

            if (';' === substr(rtrim($line), -1)) {
                if (!$verify_import || !self::should_skip_verify_statement($buffer)) {
                    DBack_Database::exec($buffer);
                    $executed++;
                }
                $buffer = '';
            }
        }

        if ('' !== trim($buffer)) {
            if (!$verify_import || !self::should_skip_verify_statement($buffer)) {
                DBack_Database::exec($buffer);
                $executed++;
            }
        }

        return $executed;
    }

    private static function is_verify_database($database) {
        return 0 === strpos((string) $database, 'dback_verify_');
    }

    private static function should_skip_verify_statement($sql) {
        $sql = trim((string) $sql);
        if ('' === $sql) {
            return true;
        }
        if (preg_match('/^CREATE\s+DATABASE/i', $sql)) {
            return true;
        }
        if (preg_match('/^DROP\s+DATABASE/i', $sql)) {
            return true;
        }
        if (preg_match('/^USE\s+/i', $sql)) {
            return true;
        }
        return false;
    }
}
