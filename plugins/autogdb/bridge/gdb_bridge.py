# -*- coding: utf-8 -*-
# SPDX-License-Identifier: MIT
from __future__ import print_function

import inspect
import json
import os
import socket
import threading
import sys

# Python 2/3 compatibility for string types
try:
    string_types = (basestring,)  # Python 2
except NameError:
    string_types = (str,)  # Python 3

try:
    import gdb  # type: ignore[import-untyped]
except ImportError:
    gdb = None  # type: ignore[assignment]


def _fail_outside_gdb():
    print(
        "gdb_bridge.py must be loaded inside GDB, e.g.:\n"
        "  (gdb) python-exec-file /path/to/gdb_bridge.py\n"
        "Running with system Python only checks that this file is readable (exit 2).",
        file=sys.stderr,
    )
    sys.exit(2)


def _install_auto_gdb_command():

    class AutoGdb(gdb.Command):
        """GDB ``autogdb`` command: ``listen`` / ``stop`` are parsed from the argument string.

        One listener at a time; listener state lives on this command instance.

        Usage::

            autogdb listen <socket_path> [-s]
            autogdb stop

        * ``listen`` — MCP clients connect via the Unix socket.
        * By default, MCP command CLI output is mirrored to this GDB terminal.
        * ``-s`` — silent; do not mirror MCP output to the GDB terminal.
        """

        def __init__(self):
            super(AutoGdb, self).__init__(
                "autogdb",
                gdb.COMMAND_USER,
                gdb.COMPLETE_FILENAME,
            )
            self.thread = None
            self.server_sock = None
            self.listen_path = None
            self.stop_event = threading.Event()
            self.gdb_cmd_lock = threading.Lock()
            # When True (default), mirror MCP command CLI output to the GDB TTY; ``listen -s`` turns it off.
            self.mirror_mcp_output_to_tty = True

        def _execute_cli_on_gdb_thread(self, cli_cmd, timeout_sec):
            ev = threading.Event()
            holder = {"out": "", "err": None}

            def job():
                try:
                    captured = gdb.execute(cli_cmd, to_string=True) or ""
                    holder["out"] = captured
                    if self.mirror_mcp_output_to_tty and captured:
                        gdb.write(captured, gdb.STDOUT)
                except gdb.error as e:  # type: ignore[attr-defined]
                    holder["err"] = str(e)
                except Exception as e:  # noqa: BLE001
                    holder["err"] = str(e)
                finally:
                    ev.set()

            gdb.post_event(job)

            timed_out = not ev.wait(timeout_sec)
            if timed_out:
                # gdb.execute runs on GDB's main thread; we cannot cancel the Python
                # closure directly. gdb.interrupt() (GDB 14+, thread-safe) asks GDB to
                # abort the current operation like Ctrl-C. Older GDB: only wait for finish.
                intr = getattr(gdb, "interrupt", None)
                if callable(intr):
                    try:
                        intr()
                    except Exception:  # noqa: BLE001 — best-effort; still sync below
                        pass
                ev.wait()
                return "", "timeout waiting for gdb response"

            if holder["err"]:
                return holder["out"] or "", str(holder["err"])
            return str(holder["out"]), None

        def handle_client(self, conn):
            # Python 2/3 compatible: wrap socket for text mode
            if sys.version_info[0] >= 3:
                conn_file = conn.makefile("r", encoding="utf-8", newline="\n")
                out_file = conn.makefile("w", encoding="utf-8", newline="\n")
            else:
                import io
                conn_file = io.open(conn.fileno(), "r", encoding="utf-8", newline="\n", closefd=False)
                out_file = io.open(conn.fileno(), "w", encoding="utf-8", newline="\n", closefd=False)
            try:
                for line in conn_file:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        req = json.loads(line)
                    except ValueError as e:
                        out_file.write(
                            json.dumps({"output": "", "error": "invalid json: {}".format(e)}) + "\n"
                        )
                        out_file.flush()
                        continue

                    cli_cmd = req.get("command")
                    if not isinstance(cli_cmd, string_types):
                        out_file.write(
                            json.dumps(
                                {
                                    "output": "",
                                    "error": "missing or invalid 'command' string",
                                }
                            )
                            + "\n"
                        )
                        out_file.flush()
                        continue

                    try:
                        timeout_sec = float(req.get("timeout", 15.0))
                    except (TypeError, ValueError):
                        timeout_sec = 15.0
                    if timeout_sec < 0:
                        timeout_sec = 0.0

                    with self.gdb_cmd_lock:
                        out_text, err = self._execute_cli_on_gdb_thread(
                            cli_cmd, timeout_sec
                        )

                    out_file.write(json.dumps({"output": out_text, "error": err}) + "\n")
                    out_file.flush()
            except (IOError, OSError):
                pass
            finally:
                try:
                    conn_file.close()
                except OSError:
                    pass
                try:
                    out_file.close()
                except OSError:
                    pass
                try:
                    conn.close()
                except OSError:
                    pass

        def serve_loop(self, sock):
            sock.settimeout(1.0)
            while not self.stop_event.is_set():
                try:
                    conn, _ = sock.accept()
                except socket.timeout:
                    continue
                except OSError as e:
                    gdb.write("autogdb: accept failed: {}\n".format(e), gdb.STDERR)
                    break
                if self.stop_event.is_set():
                    try:
                        conn.close()
                    except OSError:
                        pass
                    break
                t = threading.Thread(target=self.handle_client, args=(conn,))
                t.daemon = True
                t.start()

        def _write_command_doc(self):
            doc = inspect.getdoc(type(self))
            if doc:
                gdb.write(doc + "\n", gdb.STDERR)

        def invoke(self, arg, from_tty):
            parts = arg.split()
            if not parts:
                self._write_command_doc()
                return

            sub = parts[0].lower()
            rest = parts[1:]
            if sub == "listen":
                self._invoke_listen(rest)
            elif sub == "stop":
                self._invoke_stop(rest)
            else:
                gdb.write(
                    "autogdb: unknown subcommand {!r} (expected listen or stop)\n".format(parts[0]),
                    gdb.STDERR,
                )
                self._write_command_doc()

        def _invoke_listen(self, parts):
            if not parts:
                self._write_command_doc()
                return

            if len(parts) > 2:
                gdb.write(
                    "autogdb listen: too many arguments (expected <socket_path> [-s])\n",
                    gdb.STDERR,
                )
                self._write_command_doc()
                return

            path = parts[0].strip()
            if len(parts) == 2:
                if parts[1] != "-s":
                    gdb.write(
                        "autogdb listen: unknown argument {!r} (expected -s)\n".format(parts[1]),
                        gdb.STDERR,
                    )
                    self._write_command_doc()
                    return
                self.mirror_mcp_output_to_tty = False
            else:
                self.mirror_mcp_output_to_tty = True

            if not path:
                self._write_command_doc()
                return

            if self.thread is not None and self.thread.is_alive():
                gdb.write(
                    "autogdb listen: already listening; run autogdb stop first\n",
                    gdb.STDERR,
                )
                return

            self.stop_event.clear()
            try:
                if os.path.exists(path):
                    os.unlink(path)
            except OSError as e:
                gdb.write(
                    "autogdb listen: could not remove existing socket: {}\n".format(e), gdb.STDERR
                )
                return

            sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            try:
                sock.bind(path)
                sock.listen(8)
                os.chmod(path, 0o666)  # allow all users to read/write
            except OSError as e:
                gdb.write("autogdb listen: bind failed: {}\n".format(e), gdb.STDERR)
                try:
                    sock.close()
                except OSError:
                    pass
                return

            self.server_sock = sock
            self.listen_path = path
            self.thread = threading.Thread(
                target=self.serve_loop, args=(sock,)
            )
            self.thread.daemon = True
            self.thread.start()
            mode = "echo to TTY on" if self.mirror_mcp_output_to_tty else "echo to TTY off"
            gdb.write("autogdb listen: listening on {} ({})\n".format(path, mode), gdb.STDOUT)

        def _invoke_stop(self, parts):
            if parts:
                gdb.write("autogdb stop: unexpected arguments\n", gdb.STDERR)
                self._write_command_doc()
                return

            if self.thread is None or not self.thread.is_alive():
                gdb.write("autogdb stop: not listening\n", gdb.STDERR)
                return

            self.stop_event.set()
            if self.server_sock is not None:
                try:
                    self.server_sock.close()
                except OSError as e:
                    gdb.write(
                        "autogdb stop: close server socket failed: {}\n".format(e), gdb.STDERR
                    )
            self.thread.join()
            self.thread = None
            self.server_sock = None

            p = self.listen_path
            self.listen_path = None
            if p and os.path.exists(p):
                try:
                    os.unlink(p)
                except OSError:
                    pass

            gdb.write("autogdb stop: stopped\n", gdb.STDOUT)
            self.mirror_mcp_output_to_tty = True


    AutoGdb()


if gdb is None:
    if __name__ == "__main__":
        _fail_outside_gdb()
else:
    _install_auto_gdb_command()
