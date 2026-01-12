package dev.unqueryvet

import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.Configurable
import com.intellij.openapi.ui.TextFieldWithBrowseButton
import com.intellij.ui.components.JBCheckBox
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import javax.swing.JComponent
import javax.swing.JPanel
import javax.swing.JSpinner
import javax.swing.SpinnerNumberModel

/**
 * Settings UI for unqueryvet plugin.
 */
class UnqueryvetConfigurable : Configurable {

    private var mainPanel: JPanel? = null
    private var enabledCheckBox: JBCheckBox? = null
    private var binaryPathField: TextFieldWithBrowseButton? = null
    private var n1DetectionCheckBox: JBCheckBox? = null
    private var sqliDetectionCheckBox: JBCheckBox? = null
    private var complexityCheckBox: JBCheckBox? = null
    private var complexitySpinner: JSpinner? = null
    private var autoFixCheckBox: JBCheckBox? = null
    private var excludePatternsField: JBTextField? = null

    override fun getDisplayName(): String = "unqueryvet"

    override fun createComponent(): JComponent? {
        enabledCheckBox = JBCheckBox("Enable unqueryvet")

        binaryPathField = TextFieldWithBrowseButton().apply {
            addBrowseFolderListener(
                "Select unqueryvet Binary",
                "Choose the path to the unqueryvet executable",
                null,
                FileChooserDescriptorFactory.createSingleFileDescriptor()
            )
        }

        n1DetectionCheckBox = JBCheckBox("Enable N+1 query detection")
        sqliDetectionCheckBox = JBCheckBox("Enable SQL injection detection")
        complexityCheckBox = JBCheckBox("Enable query complexity analysis")
        complexitySpinner = JSpinner(SpinnerNumberModel(25, 1, 100, 5))
        autoFixCheckBox = JBCheckBox("Auto-fix on save")
        excludePatternsField = JBTextField()

        mainPanel = FormBuilder.createFormBuilder()
            .addComponent(enabledCheckBox!!)
            .addSeparator()
            .addLabeledComponent(JBLabel("Binary path:"), binaryPathField!!)
            .addSeparator()
            .addComponent(JBLabel("Detection Options"))
            .addComponent(n1DetectionCheckBox!!)
            .addComponent(sqliDetectionCheckBox!!)
            .addComponent(complexityCheckBox!!)
            .addLabeledComponent(JBLabel("Complexity threshold:"), complexitySpinner!!)
            .addSeparator()
            .addComponent(autoFixCheckBox!!)
            .addLabeledComponent(JBLabel("Exclude patterns (comma-separated):"), excludePatternsField!!)
            .addComponentFillVertically(JPanel(), 0)
            .panel

        return mainPanel
    }

    override fun isModified(): Boolean {
        val settings = UnqueryvetSettings.getInstance()
        return enabledCheckBox?.isSelected != settings.enabled ||
            binaryPathField?.text != settings.binaryPath ||
            n1DetectionCheckBox?.isSelected != settings.enableN1Detection ||
            sqliDetectionCheckBox?.isSelected != settings.enableSQLiDetection ||
            complexityCheckBox?.isSelected != settings.enableComplexityAnalysis ||
            (complexitySpinner?.value as? Int) != settings.complexityThreshold ||
            autoFixCheckBox?.isSelected != settings.autoFix ||
            excludePatternsField?.text != settings.excludePatterns.joinToString(", ")
    }

    override fun apply() {
        val settings = UnqueryvetSettings.getInstance()
        settings.enabled = enabledCheckBox?.isSelected ?: true
        settings.binaryPath = binaryPathField?.text ?: "unqueryvet"
        settings.enableN1Detection = n1DetectionCheckBox?.isSelected ?: true
        settings.enableSQLiDetection = sqliDetectionCheckBox?.isSelected ?: true
        settings.enableComplexityAnalysis = complexityCheckBox?.isSelected ?: false
        settings.complexityThreshold = (complexitySpinner?.value as? Int) ?: 25
        settings.autoFix = autoFixCheckBox?.isSelected ?: false
        settings.excludePatterns = (excludePatternsField?.text ?: "")
            .split(",")
            .map { it.trim() }
            .filter { it.isNotEmpty() }
            .toMutableList()
    }

    override fun reset() {
        val settings = UnqueryvetSettings.getInstance()
        enabledCheckBox?.isSelected = settings.enabled
        binaryPathField?.text = settings.binaryPath
        n1DetectionCheckBox?.isSelected = settings.enableN1Detection
        sqliDetectionCheckBox?.isSelected = settings.enableSQLiDetection
        complexityCheckBox?.isSelected = settings.enableComplexityAnalysis
        complexitySpinner?.value = settings.complexityThreshold
        autoFixCheckBox?.isSelected = settings.autoFix
        excludePatternsField?.text = settings.excludePatterns.joinToString(", ")
    }

    override fun disposeUIResources() {
        mainPanel = null
        enabledCheckBox = null
        binaryPathField = null
        n1DetectionCheckBox = null
        sqliDetectionCheckBox = null
        complexityCheckBox = null
        complexitySpinner = null
        autoFixCheckBox = null
        excludePatternsField = null
    }
}
