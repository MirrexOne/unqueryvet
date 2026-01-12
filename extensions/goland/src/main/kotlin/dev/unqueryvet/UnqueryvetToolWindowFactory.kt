package dev.unqueryvet

import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.content.ContentFactory
import com.intellij.ui.table.JBTable
import javax.swing.*
import javax.swing.table.DefaultTableModel
import java.awt.BorderLayout

/**
 * Tool window factory for displaying unqueryvet results.
 */
class UnqueryvetToolWindowFactory : ToolWindowFactory {

    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val panel = UnqueryvetToolWindowPanel(project)
        val content = ContentFactory.getInstance().createContent(panel, "Issues", false)
        toolWindow.contentManager.addContent(content)
    }
}

/**
 * Panel displaying unqueryvet analysis results.
 */
class UnqueryvetToolWindowPanel(private val project: Project) : JPanel(BorderLayout()) {

    private val tableModel = DefaultTableModel(
        arrayOf("File", "Line", "Type", "Message"),
        0
    )
    private val table = JBTable(tableModel)

    init {
        // Toolbar
        val toolbar = JPanel()
        toolbar.layout = BoxLayout(toolbar, BoxLayout.X_AXIS)

        val runButton = JButton("Run Analysis")
        runButton.addActionListener { runAnalysis() }
        toolbar.add(runButton)

        val clearButton = JButton("Clear")
        clearButton.addActionListener { clearResults() }
        toolbar.add(clearButton)

        toolbar.add(Box.createHorizontalGlue())

        val settingsButton = JButton("Settings")
        settingsButton.addActionListener { openSettings() }
        toolbar.add(settingsButton)

        add(toolbar, BorderLayout.NORTH)

        // Results table
        table.setShowGrid(true)
        table.autoResizeMode = JTable.AUTO_RESIZE_LAST_COLUMN
        table.columnModel.getColumn(0).preferredWidth = 200
        table.columnModel.getColumn(1).preferredWidth = 50
        table.columnModel.getColumn(2).preferredWidth = 100
        table.columnModel.getColumn(3).preferredWidth = 400

        // Double-click to navigate
        table.addMouseListener(object : java.awt.event.MouseAdapter() {
            override fun mouseClicked(e: java.awt.event.MouseEvent) {
                if (e.clickCount == 2) {
                    navigateToIssue()
                }
            }
        })

        add(JBScrollPane(table), BorderLayout.CENTER)

        // Status bar
        val statusLabel = JLabel("Ready")
        add(statusLabel, BorderLayout.SOUTH)
    }

    private fun runAnalysis() {
        clearResults()

        // Get project base path
        val basePath = project.basePath ?: return

        // Run unqueryvet in background
        Thread {
            try {
                val settings = UnqueryvetSettings.getInstance()
                val command = mutableListOf(settings.binaryPath)

                if (settings.enableN1Detection) {
                    command.add("-n1")
                }
                if (settings.enableSQLiDetection) {
                    command.add("-sqli")
                }

                command.add("$basePath/...")

                val process = ProcessBuilder(command)
                    .directory(java.io.File(basePath))
                    .redirectErrorStream(true)
                    .start()

                val reader = java.io.BufferedReader(java.io.InputStreamReader(process.inputStream))
                val pattern = java.util.regex.Pattern.compile("(\\S+):(\\d+):(\\d+):\\s+(.*)")

                reader.forEachLine { line ->
                    val matcher = pattern.matcher(line)
                    if (matcher.matches()) {
                        SwingUtilities.invokeLater {
                            addResult(
                                matcher.group(1),
                                matcher.group(2),
                                determineType(matcher.group(4)),
                                matcher.group(4)
                            )
                        }
                    }
                }

                process.waitFor()

            } catch (e: Exception) {
                SwingUtilities.invokeLater {
                    JOptionPane.showMessageDialog(
                        this,
                        "Error running unqueryvet: ${e.message}",
                        "Error",
                        JOptionPane.ERROR_MESSAGE
                    )
                }
            }
        }.start()
    }

    private fun determineType(message: String): String {
        return when {
            message.contains("SELECT *", ignoreCase = true) -> "SELECT *"
            message.contains("N+1", ignoreCase = true) -> "N+1"
            message.contains("injection", ignoreCase = true) -> "SQLi"
            message.contains("complexity", ignoreCase = true) -> "Complexity"
            else -> "Warning"
        }
    }

    private fun addResult(file: String, line: String, type: String, message: String) {
        tableModel.addRow(arrayOf(file, line, type, message))
    }

    private fun clearResults() {
        tableModel.rowCount = 0
    }

    private fun navigateToIssue() {
        val row = table.selectedRow
        if (row < 0) return

        val file = tableModel.getValueAt(row, 0) as String
        val line = (tableModel.getValueAt(row, 1) as String).toIntOrNull() ?: return

        // Navigate to file:line
        val virtualFile = com.intellij.openapi.vfs.LocalFileSystem.getInstance()
            .findFileByPath(file)

        if (virtualFile != null) {
            com.intellij.openapi.fileEditor.OpenFileDescriptor(project, virtualFile, line - 1, 0)
                .navigate(true)
        }
    }

    private fun openSettings() {
        com.intellij.openapi.options.ShowSettingsUtil.getInstance()
            .showSettingsDialog(project, "unqueryvet")
    }
}
