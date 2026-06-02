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
            $statements = self::split_statements($sql);
            if (empty($statements)) {
                throw new InvalidArgumentException('SQL query is required.');
            }

            if (1 === count($statements)) {
                $result = self::run_on_active_connection($statements[0]);
                $result['database'] = $selected_database;

                return $result;
            }

            $results = array();
            foreach ($statements as $statement) {
                $results[] = self::run_on_active_connection($statement);
            }

            return array(
                'success' => true,
                'type' => 'batch',
                'statements_executed' => count($results),
                'statements' => $results,
                'driver' => DBack_Database::driver(),
                'database' => $selected_database,
            );
        });
    }

    /**
     * Split a SQL script into individual statements (semicolon-delimited).
     *
     * @param string $sql
     * @return string[]
     */
    public static function split_statements($sql) {
        $sql = trim((string) $sql);
        if ('' === $sql) {
            return array();
        }

        $statements = array();
        $buffer = '';
        $lines = preg_split('/\R/', $sql);

        foreach ($lines as $line) {
            $trimmed = trim($line);
            if ('' === $trimmed && '' === trim($buffer)) {
                continue;
            }
            if ('' !== $trimmed && (0 === strpos($trimmed, '--') || 0 === strpos($trimmed, '#'))) {
                continue;
            }

            $buffer .= $line . "\n";
            if (';' === substr(rtrim($line), -1)) {
                $statement = trim($buffer);
                if ('' !== $statement) {
                    $statements[] = $statement;
                }
                $buffer = '';
            }
        }

        $tail = trim($buffer);
        if ('' !== $tail) {
            $statements[] = $tail;
        }

        return $statements;
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
