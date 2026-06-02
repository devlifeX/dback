<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Exporter_Mysqli {

    const INSERT_BATCH_SIZE = 100;

    /**
     * Stream a gzip-compressed SQL dump using WordPress mysqli connection.
     *
     * @throws Exception
     */
    public static function stream_gzip() {
        @set_time_limit(0);
        if (function_exists('ini_set')) {
            @ini_set('memory_limit', '512M');
        }

        self::send_download_headers();

        $wpdb = DBack_Database::wpdb();
        $stream = new DBack_Gzip_Stream('php://output');

        try {
            $stream->write_line('-- DBack DB Tools dump (mysqli fallback)');
            $stream->write_line('-- Host: ' . DB_HOST . '    Database: ' . DB_NAME);
            $stream->write_line('SET NAMES utf8mb4;');
            $stream->write_line('SET FOREIGN_KEY_CHECKS=0;');
            $stream->write_line("SET SQL_MODE='NO_AUTO_VALUE_ON_ZERO';");
            $stream->write_line('SET time_zone = "+00:00";');

            $tables = $wpdb->get_results('SHOW FULL TABLES', ARRAY_N);
            DBack_Database::assert_no_db_error($wpdb);

            foreach ($tables as $table_info) {
                $table_name = $table_info[0];
                $table_type = isset($table_info[1]) ? $table_info[1] : 'BASE TABLE';

                if ('BASE TABLE' === $table_type) {
                    self::dump_table($wpdb, $stream, $table_name);
                    continue;
                }

                if ('VIEW' === $table_type) {
                    self::dump_view($wpdb, $stream, $table_name);
                }
            }

            $stream->write_line('SET FOREIGN_KEY_CHECKS=1;');
        } finally {
            $stream->close();
        }

        exit;
    }

    private static function send_download_headers() {
        while (ob_get_level() > 0) {
            ob_end_clean();
        }

        if (function_exists('apache_setenv')) {
            @apache_setenv('no-gzip', '1');
        }

        @ini_set('zlib.output_compression', 'Off');

        $filename = 'dback-export-' . gmdate('Y-m-d-His') . '.sql.gz';

        header('Content-Type: application/gzip');
        header('Content-Disposition: attachment; filename="' . $filename . '"');
        header('Cache-Control: no-store, no-cache, must-revalidate');
        header('Pragma: no-cache');
        header('X-Content-Type-Options: nosniff');
    }

    /**
     * @param string $name
     * @return string
     */
    private static function quote_identifier($name) {
        return '`' . str_replace('`', '``', $name) . '`';
    }

    /**
     * @param wpdb $wpdb
     * @param DBack_Gzip_Stream $stream
     * @param string $table
     */
    private static function dump_table($wpdb, $stream, $table) {
        $quoted_table = self::quote_identifier($table);
        $create = $wpdb->get_row('SHOW CREATE TABLE ' . $quoted_table, ARRAY_N);
        DBack_Database::assert_no_db_error($wpdb);

        if (empty($create[1])) {
            throw new RuntimeException('Unable to read table schema for ' . $table);
        }

        $stream->write_line('');
        $stream->write_line('-- Table structure for `' . $table . '`');
        $stream->write_line('DROP TABLE IF EXISTS ' . $quoted_table . ';');
        $stream->write_line($create[1] . ';');
        $stream->write_line('LOCK TABLES ' . $quoted_table . ' WRITE;');
        $stream->write_line('/*!40000 ALTER TABLE ' . $quoted_table . ' DISABLE KEYS */;');

        self::dump_table_rows($wpdb, $stream, $table);

        $stream->write_line('/*!40000 ALTER TABLE ' . $quoted_table . ' ENABLE KEYS */;');
        $stream->write_line('UNLOCK TABLES;');
    }

    /**
     * @param wpdb $wpdb
     * @param DBack_Gzip_Stream $stream
     * @param string $table
     */
    private static function dump_table_rows($wpdb, $stream, $table) {
        $columns = $wpdb->get_col('DESCRIBE ' . self::quote_identifier($table), 0);
        DBack_Database::assert_no_db_error($wpdb);

        if (empty($columns)) {
            return;
        }

        if ($wpdb->dbh instanceof mysqli) {
            self::dump_table_rows_unbuffered($wpdb, $stream, $table, $columns);
            return;
        }

        self::dump_table_rows_batched($wpdb, $stream, $table, $columns);
    }

    /**
     * @param wpdb $wpdb
     * @param DBack_Gzip_Stream $stream
     * @param string $table
     * @param array<int,string> $columns
     */
    private static function dump_table_rows_unbuffered($wpdb, $stream, $table, $columns) {
        $quoted_table = self::quote_identifier($table);
        $result = mysqli_query($wpdb->dbh, 'SELECT * FROM ' . $quoted_table, MYSQLI_USE_RESULT);
        if (false === $result) {
            throw new RuntimeException($wpdb->last_error ?: 'Unable to read table data.');
        }

        $batch = array();

        try {
            while ($row = mysqli_fetch_assoc($result)) {
                $batch[] = $row;
                if (count($batch) >= self::INSERT_BATCH_SIZE) {
                    self::write_insert_batch($stream, $table, $columns, $batch, $wpdb);
                    $batch = array();
                }
            }
        } finally {
            mysqli_free_result($result);
        }

        if (!empty($batch)) {
            self::write_insert_batch($stream, $table, $columns, $batch, $wpdb);
        }
    }

    /**
     * @param wpdb $wpdb
     * @param DBack_Gzip_Stream $stream
     * @param string $table
     * @param array<int,string> $columns
     */
    private static function dump_table_rows_batched($wpdb, $stream, $table, $columns) {
        $quoted_table = self::quote_identifier($table);
        $offset = 0;

        while (true) {
            $rows = $wpdb->get_results(
                'SELECT * FROM ' . $quoted_table . ' LIMIT ' . (int) $offset . ', ' . self::INSERT_BATCH_SIZE,
                ARRAY_A
            );
            DBack_Database::assert_no_db_error($wpdb);

            if (empty($rows)) {
                break;
            }

            self::write_insert_batch($stream, $table, $columns, $rows, $wpdb);
            $offset += self::INSERT_BATCH_SIZE;
        }
    }

    /**
     * @param wpdb $wpdb
     * @param DBack_Gzip_Stream $stream
     * @param string $view
     */
    private static function dump_view($wpdb, $stream, $view) {
        $quoted_view = self::quote_identifier($view);
        $create = $wpdb->get_row('SHOW CREATE VIEW ' . $quoted_view, ARRAY_N);
        DBack_Database::assert_no_db_error($wpdb);

        if (empty($create[1])) {
            return;
        }

        $stream->write_line('');
        $stream->write_line('-- View structure for `' . $view . '`');
        $stream->write_line('DROP VIEW IF EXISTS ' . $quoted_view . ';');
        $stream->write_line($create[1] . ';');
    }

    /**
     * @param DBack_Gzip_Stream $stream
     * @param string $table
     * @param array<int,string> $columns
     * @param array<int,array<string,mixed>> $rows
     * @param wpdb $wpdb
     */
    private static function write_insert_batch($stream, $table, $columns, $rows, $wpdb) {
        $quoted_table = self::quote_identifier($table);
        $column_list = '`' . implode('`,`', $columns) . '`';
        $values = array();

        foreach ($rows as $row) {
            $row_values = array();
            foreach ($columns as $column) {
                $row_values[] = self::sql_value($wpdb, isset($row[$column]) ? $row[$column] : null);
            }
            $values[] = '(' . implode(',', $row_values) . ')';
        }

        $sql = 'INSERT INTO ' . $quoted_table . ' (' . $column_list . ') VALUES ' . implode(',', $values) . ';';
        $stream->write_line($sql);
    }

    /**
     * @param wpdb $wpdb
     * @param mixed $value
     * @return string
     */
    private static function sql_value($wpdb, $value) {
        if (null === $value) {
            return 'NULL';
        }

        if (is_bool($value)) {
            return $value ? '1' : '0';
        }

        if (is_int($value) || is_float($value)) {
            return (string) $value;
        }

        return "'" . $wpdb->_real_escape((string) $value) . "'";
    }
}
