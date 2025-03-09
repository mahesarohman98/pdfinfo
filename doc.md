# PDF

PDF is a data structure built from Values, each of which has
one of the following kinds:

| Kind    | Description                         | Example                             |
|---------|-------------------------------------|-------------------------------------|
| Null    | Null object                         | `null`                              |
| Integer | For an integer                      | `123`                               |
| Real    | For a floating-point number         | `3.14`                              |
| Bool    | For a boolean value                 | `true` or `false`                   |
| Name    | For a name constant                 | `/Title`                            |
| String  | For a string constant               | `(Hello, PDF!)`, `<48656C6C6F>`     |
| Dict    | Dictionary of name-value pairs      | `<< /Type /Catalog /Pages 2 0 R >>` |
| Array   | An array of values                  | `[1 2 3]` or `[ /A /B /C ]`         |
| Stream  | A binary data stream (large data)   | `stream ... endstream`              |

This package focus on extracting metadata out of a pdf files.

The method for extracting metadata varies depending on the PDF version.

--- 

# Where metadata is stored in a PDF

Metadata typically stored in:

- **The `/Info` dictionary** (PDF 1.0–1.4)
- **The XMP (Extensible Metadata Platform) stream** (PDF 1.5+)
- **Cross-reference tables (`xref`) or object streams (`/XRef` objects)** (PDF 1.5+)

---

# How to extract metadata

## PDF 1.0 - 1.4 :

- ### Using Info dictionary
    - Locate the **trailer** section at the end of PDF.
    - Find the `/Info` key, which point to xref table `/Info 2 0 R`
    - Read the reference to extract metadata fields.
       **Example:**
       ```plaintext
       2 0 obj
       << /Title (Sample PDF) /Author (John Doe) /Producer (PDF Creator) >>
       endobj
       ```
    - Extract key value pair from the dictionary

## PDF 1.5+ :

- ### Using Metadata in XMP Stream
    - not implement yet
    - Locate `/Metadata` key in the trailer eg `/Metadata 5 0 R`.
    - Decode the xmp metadata stream.
        **Example:**
        ```plaintext
        5 0 obj
        << /Subtype /XML /Length 1234 >>
        stream
        <?xpacket ...>
        <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
            <rdf:Description rdf:about=""
                xmlns:dc="http://purl.org/dc/elements/1.1/">
                <dc:title>Sample PDF</dc:title>
                <dc:creator>John Doe</dc:creator>
            </rdf:Description>
        </rdf:RDF>
        </xpacket>
        endstream
        endobj
        ```
    - Parse the xmp metadata using xml parser.

- ### Using Comporess Cross Reference Stream
    - Locate the `/Ref` object instead of the standard `xref` object from pdf 1.0-1.4.
    - Decode the cross reference using `/W` array.
        **Example:**
        ```plaintext
        << /Type /XRef
       /Size 30000
       /W [1 3 1]
       /Index [1 1 17 1 1083 1 29978 1]
       /Info 2 0 R
        >>
        stream
        (binary data)
        endstream
        ```
    - Extract the object offsets and decompress metadata objects.

---
