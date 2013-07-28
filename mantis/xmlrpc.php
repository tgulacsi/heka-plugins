<?php
ini_set('display_startup_errors', 'On');
// -*- coding: utf-8 -*-
define('LOG_DEBUG', 10);
define('LOG_INFO', 20);
define('LOG_WARN', 30);
define('LOG_ERROR', 40);
$d_log_levels = array(LOG_DEBUG=>'DEBUG', LOG_INFO=>'INFO', LOG_WARN=>'WARN',
    LOG_ERROR=>'ERROR');

require_once('core.php');
require_once('authentication_api.php');

$g_loglevel = LOG_DEBUG;
function log_msg($level, $msg) {
    global $g_loglevel, $d_log_levels;
    if( TRUE || $level >= $g_loglevel ) {
        $fh = fopen('/var/log/mantis/'.SYS_COMPANY.'-'.SYS_FLAVOR
                    .'-xmlrpc_vv.log', 'a');
        fwrite($fh,
            strftime("%Y-%m-%dT%H:%M:%S")
            . ' ' . SYS_COMPANY . '/' . SYS_FLAVOR
            . " [" . $d_log_levels[$level] . "] $msg\n");
        fflush($fh);
        fclose($fh);
    }
}
function _mark($p_mark, $p_text='') {
    log_msg(LOG_DEBUG, $p_mark. ': '.$p_text);
}
_mark(0);
date_default_timezone_set('Europe/Budapest');
iconv_set_encoding('all', 'UTF-8');

log_msg(LOG_INFO, "user=".$_SERVER['PHP_AUTH_USER']);
log_msg(LOG_INFO, "passw=".$_SERVER['PHP_AUTH_PW']);

if( !isset($_SERVER['PHP_AUTH_USER'])
    || (false === auth_attempt_script_login( $_SERVER['PHP_AUTH_USER'],
                                             $_SERVER['PHP_AUTH_PW'] ))) {
    log_msg(LOG_INFO, '401 Unauthorized');
    header('WWW-Authenticate: Basic realm="UNO-SOFT Mantis"');
    header('HTTP/1.0 401 Unauthorized');
    echo 'Text to send if user hits Cancel button';
    exit;
}

$v_user_id = auth_get_current_user_id();
_mark(6, "uid=$v_user_id");

function format_date($datestring, $fmt_from, $fmt_to) {
    $ftime = strptime($datestring, $fmt_from);
    $ts = mktime($ftime['tm_hour'], $ftime['tm_min'], $ftime['tm_sec'], 1,
                 $ftime['tm_yday'] + 1, $ftime['tm_year'] + 1900
                 );
    return strftime($fmt_to, $ts);
}

$old_error_handler = null;
function myErrorHandler($errno, $errstr, $errfile, $errline) {
    global $old_error_handler;
    log_msg(LOG_ERROR, "<$errfile:$errline> $errno: $errstr");
    if( $errno == ERROR_FILE_DUPLICATE ) {
        return true;
    } else {
        if( $old_error_handler != null )
            return $old_error_handler($errno, $errstr, $errfile, $errline);
        else
            return false;
    }
}
set_error_handler("myErrorHandler");
_mark(8);

/* method implementation */
function new_issue($method_name, $params, $user_data) {
    global $v_user_id;
    $v_errcode = 98; $v_errmsg = '';

    # var_dump(func_get_args('impl'));
    if( count($params) == 1) $params = $params[0];
    log_msg(LOG_INFO, 'params='.var_export($params, true));

    $v_errcode = 98; //$v_errmsg = 'unknown error';
    if( !(array_key_exists('project_name', $params)
        && array_key_exists('category', $params)
        && array_key_exists('summary', $params)
        && array_key_exists('description', $params)) ) {
        $v_errcode = 1; $v_errmsg = 'needed params: project_name, category, summary, description';
    } else {
        ##
        ## paraméterek feldolgozása
        ##
        require_once('project_api.php');
        $t_proj_name = $params['project_name'];
        $v_project_id = project_get_id_by_name( $t_proj_name, false );
        if( $v_project_id <= 0 && is_numeric($t_proj_name)
                && project_exists( int($t_proj_name) ) ) {
            $v_project_id = int($t_proj_name);
        }
        if( $v_project_id <= 0 ) {
            $v_errcode = 2; $v_errmsg = "project name $t_proj_name not found!";
        } else {
            $t_cat_name = $params['category'];
            require_once('category_api.php');
            $v_category_id = category_get_id_by_name( $t_cat_name, $v_project_id, false );
            if ( $v_category_id <= 0 && is_numeric($t_cat_name) && category_exists( int($t_cat_name) ) ) {
                $v_category_id = int($t_cat_name);
            }
            if( $v_category_id <= 0 ) {
                $v_errcode = 3; $v_errmsg = "category name $t_cat_name not found!";
            } else {
                require_once('bug_api.php');
                $v_bug = new BugData;
                $v_bug->project_id = $v_project_id;
                $v_bug->category_id = $v_category_id;
                $v_bug->summary = $params['summary'];
                $v_bug->description = $params['description'];
                $v_bug_id = $v_bug->create();
                $v_errcode = 0; $v_errmsg = "bug $v_bug_id created";
            }
        }
    }
    log_msg(LOG_INFO, "errcode=$v_errcode, errmsg=$v_errmsg");
    return array('errcode'=>$v_errcode, 'errmsg'=>$v_errmsg,
        #'input'=>$params
        );
}


/* create server */
$server = xmlrpc_server_create();
xmlrpc_server_register_method($server, 'new_issue', 'new_issue');
_mark(9, $server);

if ($response = xmlrpc_server_call_method($server,
                                          $HTTP_RAW_POST_DATA, null)) {
    header('Content-Type: text/xml');
    echo $response;
}
_mark(10);

auth_logout();
xmlrpc_server_destroy($server);

