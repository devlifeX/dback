<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Query_Runner {

    /**
     * @param string $sql
     * @param string $target_database Optional database name; empty uses the WordPress default database.
     * @return array<string,mixed>
     * @throws Exception
     */
    public static function run($sql, $target_database = '') {
        return DBack_Database::with_target_database($target_database, function ($selected_database) use ($sql) {
            $result = self::run_on_active_connection($sql);
            $result['database'] = $selected_database;

            return $result;
        });
    }

    /**
     * @param string $sql
     * @return array<string,mixed>
     * @throws Exception
     */
    private static function run_on_active_connection($sql) {
        $sql = trim((string) $sql);
        if ('' === $sql) {
            throw new InvalidArgumentException('SQL query is required.');
        }

        @set_time_limit(0);

        $type = self::detect_query_type($sql);

        if (self::is_result_query($type)) {
            $rows = DBack_Database::query_rows($sql);
            $columns = array();

            if (!empty($rows)) {
                $columns = array_keys($rows[0]);
            }

            return array(
                'success' => true,
                'type' => 'result',
                'query_type' => $type,
                'columns' => $columns,
                'rows' => $rows,
                'row_count' => count($rows),
                'driver' => DBack_Database::driver(),
            );
        }

        $affected = DBack_Database::exec($sql);

        return array(
            'success' => true,
            'type' => 'command',
            'query_type' => $type,
            'affected_rows' => $affected,
            'driver' => DBack_Database::driver(),
        );
    }

    private static function detect_query_type($sql) {
        if (preg_match('/^\s*([A-Za-z]+)/', $sql, $matches)) {
            return strtoupper($matches[1]);
        }

        return 'UNKNOWN';
    }

    private static function is_result_query($type) {
        return in_array($type, array('SELECT', 'SHOW', 'DESCRIBE', 'DESC', 'EXPLAIN', 'WITH'), true);
    }
}
