package common

//revive:disable
const (
	TSDB_DATA_TYPE_NULL       = 0  // 1 bytes
	TSDB_DATA_TYPE_BOOL       = 1  // 1 bytes
	TSDB_DATA_TYPE_TINYINT    = 2  // 1 byte
	TSDB_DATA_TYPE_SMALLINT   = 3  // 2 bytes
	TSDB_DATA_TYPE_INT        = 4  // 4 bytes
	TSDB_DATA_TYPE_BIGINT     = 5  // 8 bytes
	TSDB_DATA_TYPE_FLOAT      = 6  // 4 bytes
	TSDB_DATA_TYPE_DOUBLE     = 7  // 8 bytes
	TSDB_DATA_TYPE_BINARY     = 8  // string
	TSDB_DATA_TYPE_TIMESTAMP  = 9  // 8 bytes
	TSDB_DATA_TYPE_NCHAR      = 10 // unicode string
	TSDB_DATA_TYPE_UTINYINT   = 11 // 1 byte
	TSDB_DATA_TYPE_USMALLINT  = 12 // 2 bytes
	TSDB_DATA_TYPE_UINT       = 13 // 4 bytes
	TSDB_DATA_TYPE_UBIGINT    = 14 // 8 bytes
	TSDB_DATA_TYPE_JSON       = 15
	TSDB_DATA_TYPE_VARBINARY  = 16
	TSDB_DATA_TYPE_DECIMAL    = 17
	TSDB_DATA_TYPE_BLOB       = 18
	TSDB_DATA_TYPE_MEDIUMBLOB = 19
	TSDB_DATA_TYPE_GEOMETRY   = 20
)

const (
	TSDB_DATA_TYPE_NULL_Str      = "NULL"
	TSDB_DATA_TYPE_BOOL_Str      = "BOOL"
	TSDB_DATA_TYPE_TINYINT_Str   = "TINYINT"
	TSDB_DATA_TYPE_SMALLINT_Str  = "SMALLINT"
	TSDB_DATA_TYPE_INT_Str       = "INT"
	TSDB_DATA_TYPE_BIGINT_Str    = "BIGINT"
	TSDB_DATA_TYPE_FLOAT_Str     = "FLOAT"
	TSDB_DATA_TYPE_DOUBLE_Str    = "DOUBLE"
	TSDB_DATA_TYPE_BINARY_Str    = "VARCHAR"
	TSDB_DATA_TYPE_TIMESTAMP_Str = "TIMESTAMP"
	TSDB_DATA_TYPE_NCHAR_Str     = "NCHAR"
	TSDB_DATA_TYPE_UTINYINT_Str  = "TINYINT UNSIGNED"
	TSDB_DATA_TYPE_USMALLINT_Str = "SMALLINT UNSIGNED"
	TSDB_DATA_TYPE_UINT_Str      = "INT UNSIGNED"
	TSDB_DATA_TYPE_UBIGINT_Str   = "BIGINT UNSIGNED"
	TSDB_DATA_TYPE_JSON_Str      = "JSON"
	TSDB_DATA_TYPE_VARBINARY_Str = "VARBINARY"
	TSDB_DATA_TYPE_GEOMETRY_Str  = "GEOMETRY"
)

var TypeNameMap = map[int]string{
	TSDB_DATA_TYPE_NULL:      TSDB_DATA_TYPE_NULL_Str,
	TSDB_DATA_TYPE_BOOL:      TSDB_DATA_TYPE_BOOL_Str,
	TSDB_DATA_TYPE_TINYINT:   TSDB_DATA_TYPE_TINYINT_Str,
	TSDB_DATA_TYPE_SMALLINT:  TSDB_DATA_TYPE_SMALLINT_Str,
	TSDB_DATA_TYPE_INT:       TSDB_DATA_TYPE_INT_Str,
	TSDB_DATA_TYPE_BIGINT:    TSDB_DATA_TYPE_BIGINT_Str,
	TSDB_DATA_TYPE_FLOAT:     TSDB_DATA_TYPE_FLOAT_Str,
	TSDB_DATA_TYPE_DOUBLE:    TSDB_DATA_TYPE_DOUBLE_Str,
	TSDB_DATA_TYPE_BINARY:    TSDB_DATA_TYPE_BINARY_Str,
	TSDB_DATA_TYPE_TIMESTAMP: TSDB_DATA_TYPE_TIMESTAMP_Str,
	TSDB_DATA_TYPE_NCHAR:     TSDB_DATA_TYPE_NCHAR_Str,
	TSDB_DATA_TYPE_UTINYINT:  TSDB_DATA_TYPE_UTINYINT_Str,
	TSDB_DATA_TYPE_USMALLINT: TSDB_DATA_TYPE_USMALLINT_Str,
	TSDB_DATA_TYPE_UINT:      TSDB_DATA_TYPE_UINT_Str,
	TSDB_DATA_TYPE_UBIGINT:   TSDB_DATA_TYPE_UBIGINT_Str,
	TSDB_DATA_TYPE_JSON:      TSDB_DATA_TYPE_JSON_Str,
	TSDB_DATA_TYPE_VARBINARY: TSDB_DATA_TYPE_VARBINARY_Str,
	TSDB_DATA_TYPE_GEOMETRY:  TSDB_DATA_TYPE_GEOMETRY_Str,
}

var NameTypeMap = map[string]int{
	TSDB_DATA_TYPE_NULL_Str:      TSDB_DATA_TYPE_NULL,
	TSDB_DATA_TYPE_BOOL_Str:      TSDB_DATA_TYPE_BOOL,
	TSDB_DATA_TYPE_TINYINT_Str:   TSDB_DATA_TYPE_TINYINT,
	TSDB_DATA_TYPE_SMALLINT_Str:  TSDB_DATA_TYPE_SMALLINT,
	TSDB_DATA_TYPE_INT_Str:       TSDB_DATA_TYPE_INT,
	TSDB_DATA_TYPE_BIGINT_Str:    TSDB_DATA_TYPE_BIGINT,
	TSDB_DATA_TYPE_FLOAT_Str:     TSDB_DATA_TYPE_FLOAT,
	TSDB_DATA_TYPE_DOUBLE_Str:    TSDB_DATA_TYPE_DOUBLE,
	TSDB_DATA_TYPE_BINARY_Str:    TSDB_DATA_TYPE_BINARY,
	TSDB_DATA_TYPE_TIMESTAMP_Str: TSDB_DATA_TYPE_TIMESTAMP,
	TSDB_DATA_TYPE_NCHAR_Str:     TSDB_DATA_TYPE_NCHAR,
	TSDB_DATA_TYPE_UTINYINT_Str:  TSDB_DATA_TYPE_UTINYINT,
	TSDB_DATA_TYPE_USMALLINT_Str: TSDB_DATA_TYPE_USMALLINT,
	TSDB_DATA_TYPE_UINT_Str:      TSDB_DATA_TYPE_UINT,
	TSDB_DATA_TYPE_UBIGINT_Str:   TSDB_DATA_TYPE_UBIGINT,
	TSDB_DATA_TYPE_JSON_Str:      TSDB_DATA_TYPE_JSON,
	TSDB_DATA_TYPE_VARBINARY_Str: TSDB_DATA_TYPE_VARBINARY,
	TSDB_DATA_TYPE_GEOMETRY_Str:  TSDB_DATA_TYPE_GEOMETRY,
}