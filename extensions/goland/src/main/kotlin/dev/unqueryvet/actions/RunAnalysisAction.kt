package dev.unqueryvet.actions

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.ui.Messages
import com.intellij.openapi.wm.ToolWindowManager
import dev.unqueryvet.UnqueryvetSettings
import java.io.BufferedReader
import java.io.File
import java.io.InputStreamReader

/**
 * Action to run unqueryvet analysis on the current file or project.
 */
class RunAnalysisAction : AnAction() {

    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val file = e.getData(CommonDataKeys.VIRTUAL_FILE)

        val targetPath = file?.path ?: project.basePath ?: return

        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Running unqueryvet analysis") {
            override fun run(indicator: ProgressIndicator) {
                indicator.isIndeterminate = true
                indicator.text = "Analyzing..."

                try {
                    val settings = UnqueryvetSettings.getInstance()
                    val command = mutableListOf(settings.binaryPath)

                    if (settings.enableN1Detection) {
                        command.add("-n1")
                    }
                    if (settings.enableSQLiDetection) {
                        command.add("-sqli")
                    }

                    // Add target path
                    val target = if (file != null && !file.isDirectory) {
                        targetPath
                    } else {
                        "$targetPath/..."
                    }
                    command.add(target)

                    val process = ProcessBuilder(command)
                        .directory(File(project.basePath ?: "."))
                        .redirectErrorStream(true)
                        .start()

                    val reader = BufferedReader(InputStreamReader(process.inputStream))
                    val output = StringBuilder()
                    var issueCount = 0

                    reader.forEachLine { line ->
                        output.append(line).append("\n")
                        if (line.matches(Regex(".*:\\d+:\\d+:.*"))) {
                            issueCount++
                        }
                    }

                    process.waitFor()

                    // Show results in tool window
                    com.intellij.openapi.application.ApplicationManager.getApplication().invokeLater {
                        val toolWindow = ToolWindowManager.getInstance(project).getToolWindow("unqueryvet")
                        toolWindow?.show()

                        if (issueCount == 0) {
                            Messages.showInfoMessage(
                                project,
                                "No issues found!",
                                "unqueryvet Analysis Complete"
                            )
                        }
                    }

                } catch (ex: Exception) {
                    com.intellij.openapi.application.ApplicationManager.getApplication().invokeLater {
                        Messages.showErrorDialog(
                            project,
                            "Error running unqueryvet: ${ex.message}",
                            "Analysis Failed"
                        )
                    }
                }
            }
        })
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        e.presentation.isEnabledAndVisible = project != null
    }
}
