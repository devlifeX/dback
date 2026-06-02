<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Rest_Controller {

    public function register_routes() {
        register_rest_route(DBACK_DB_TOOLS_REST_NAMESPACE, '/export', array(
            'methods' => WP_REST_Server::READABLE,
            'callback' => array($this, 'handle_export'),
            'permission_callback' => array($this, 'check_permission'),
        ));

        register_rest_route(DBACK_DB_TOOLS_REST_NAMESPACE, '/import', array(
            'methods' => WP_REST_Server::CREATABLE,
            'callback' => array($this, 'handle_import'),
            'permission_callback' => array($this, 'check_permission'),
        ));

        register_rest_route(DBACK_DB_TOOLS_REST_NAMESPACE, '/query', array(
            'methods' => WP_REST_Server::CREATABLE,
            'callback' => array($this, 'handle_query'),
            'permission_callback' => array($this, 'check_permission'),
            'args' => array(
                'sql' => array(
                    'required' => true,
                    'type' => 'string',
                    'sanitize_callback' => 'sanitize_textarea_field',
                ),
            ),
        ));

        register_rest_route(DBACK_DB_TOOLS_REST_NAMESPACE, '/logs', array(
            array(
                'methods' => WP_REST_Server::READABLE,
                'callback' => array($this, 'handle_get_logs'),
                'permission_callback' => array($this, 'check_permission'),
            ),
            array(
                'methods' => WP_REST_Server::DELETABLE,
                'callback' => array($this, 'handle_clear_logs'),
                'permission_callback' => array($this, 'check_admin_permission'),
            ),
        ));
    }

    /**
     * @param WP_REST_Request $request
     * @return bool|WP_Error
     */
    public function check_permission($request) {
        $api_key = $request->get_header('X-DBACK-KEY');
        if (DBack_Api_Key::is_valid($api_key)) {
            return true;
        }

        if (current_user_can('manage_options')) {
            return true;
        }

        return new WP_Error(
            'dback_forbidden',
            __('You are not allowed to access DBack DB Tools.', 'dback-db-tools'),
            array('status' => 403)
        );
    }

    /**
     * @param WP_REST_Request $request
     * @return bool|WP_Error
     */
    public function check_admin_permission($request) {
        if (current_user_can('manage_options')) {
            return true;
        }

        return new WP_Error(
            'dback_forbidden',
            __('Only administrators can clear the error log.', 'dback-db-tools'),
            array('status' => 403)
        );
    }

    /**
     * @param WP_REST_Request $request
     * @return void|WP_Error
     */
    public function handle_export($request) {
        try {
            DBack_Exporter::stream_gzip();
        } catch (Throwable $exception) {
            return DBack_Error_Logger::to_wp_error('export', 'dback_export_failed', $exception);
        }
    }

    /**
     * @param WP_REST_Request $request
     * @return WP_REST_Response|WP_Error
     */
    public function handle_import($request) {
        try {
            $result = DBack_Importer::import_request_body();
            return rest_ensure_response(array(
                'success' => true,
                'message' => __('Database imported successfully.', 'dback-db-tools'),
                'statements_executed' => $result['statements_executed'],
            ));
        } catch (Throwable $exception) {
            return DBack_Error_Logger::to_wp_error('import', 'dback_import_failed', $exception);
        }
    }

    /**
     * @param WP_REST_Request $request
     * @return WP_REST_Response|WP_Error
     */
    public function handle_query($request) {
        $sql = $request->get_param('sql');

        try {
            $result = DBack_Query_Runner::run($sql);
            return rest_ensure_response($result);
        } catch (Throwable $exception) {
            return DBack_Error_Logger::to_wp_error('query', 'dback_query_failed', $exception);
        }
    }

    /**
     * @param WP_REST_Request $request
     * @return WP_REST_Response
     */
    public function handle_get_logs($request) {
        $limit = (int) $request->get_param('limit');
        if ($limit <= 0) {
            $limit = 50;
        }

        return rest_ensure_response(array(
            'success' => true,
            'entries' => DBack_Error_Logger::get_entries($limit),
            'log_file' => basename(DBack_Error_Logger::log_file_path()),
        ));
    }

    /**
     * @param WP_REST_Request $request
     * @return WP_REST_Response
     */
    public function handle_clear_logs($request) {
        DBack_Error_Logger::clear();

        return rest_ensure_response(array(
            'success' => true,
            'message' => __('Error log cleared.', 'dback-db-tools'),
        ));
    }
}
